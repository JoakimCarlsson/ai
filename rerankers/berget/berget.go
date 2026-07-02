// Package berget provides a Berget AI implementation of the
// [rerankers.Reranker] interface.
//
// Berget serves rerank models (e.g. BAAI/bge-reranker-v2-m3) over a
// Cohere/Jina-compatible POST /v1/rerank endpoint. See
// [github.com/joakimcarlsson/ai/model] for the catalog (BergetRerankerModels)
// and pricing (EUR).
package berget

import (
	"context"
	"net/http"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rerankers"
)

const defaultBaseURL = "https://api.berget.ai/v1"

// Options configures the Berget reranker client.
type Options struct {
	apiKey     string
	model      model.RerankerModel
	timeout    *time.Duration
	topK       *int
	returnDocs bool
	baseURL    string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Berget.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the reranker model.
func WithModel(m model.RerankerModel) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithTopK limits how many top-ranked documents are returned.
func WithTopK(topK int) Option {
	return func(o *Options) { o.topK = &topK }
}

// WithReturnDocuments controls whether document text is included in results.
func WithReturnDocuments(returnDocs bool) Option {
	return func(o *Options) { o.returnDocs = returnDocs }
}

// WithBaseURL points the client at a custom endpoint (defaults to
// https://api.berget.ai/v1).
func WithBaseURL(baseURL string) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// Client implements [rerankers.Reranker] against the Berget rerank API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewReranker constructs a Berget reranker client. The returned
// [rerankers.Reranker] is wrapped with [rerankers.WithTracing], so callers
// always get tracing spans and metrics.
func NewReranker(opts ...Option) rerankers.Reranker {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	baseURL := defaultBaseURL
	if options.baseURL != "" {
		baseURL = options.baseURL
	}

	return rerankers.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
	}, rerankers.TracingAttrs{
		TopK:            options.topK,
		ReturnDocuments: &options.returnDocs,
	})
}

// Model returns the configured reranker model.
func (c *Client) Model() model.RerankerModel {
	return c.options.model
}

type request struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            *int     `json:"top_n,omitempty"`
	ReturnDocuments bool     `json:"return_documents,omitempty"`
}

type response struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
		Document       *struct {
			Text string `json:"text"`
		} `json:"document,omitempty"`
	} `json:"results"`
	// vLLM-style usage; Cohere-style meta.billed_units kept as a fallback.
	Usage struct {
		TotalTokens int64 `json:"total_tokens"`
	} `json:"usage"`
	Meta struct {
		BilledUnits struct {
			SearchUnits int64 `json:"search_units"`
		} `json:"billed_units"`
	} `json:"meta"`
}

// Rerank reorders documents by relevance to the query.
func (c *Client) Rerank(
	ctx context.Context,
	query string,
	documents []string,
) (*rerankers.RerankerResponse, error) {
	reqBody := request{
		Model:           c.options.model.APIModel,
		Query:           query,
		Documents:       documents,
		TopN:            c.options.topK,
		ReturnDocuments: c.options.returnDocs,
	}

	var bergetResp response
	if err := rerankers.PostJSON(
		ctx,
		c.httpClient,
		c.baseURL+"/rerank",
		c.options.apiKey,
		reqBody,
		&bergetResp,
	); err != nil {
		return nil, err
	}

	results := make([]rerankers.RerankerResult, len(bergetResp.Results))
	for i, data := range bergetResp.Results {
		result := rerankers.RerankerResult{
			Index:          data.Index,
			RelevanceScore: data.RelevanceScore,
		}
		if data.Document != nil {
			result.Document = data.Document.Text
		}
		results[i] = result
	}

	tokens := bergetResp.Usage.TotalTokens
	if tokens == 0 {
		tokens = bergetResp.Meta.BilledUnits.SearchUnits
	}

	return &rerankers.RerankerResponse{
		Results: results,
		Usage: rerankers.RerankerUsage{
			TotalTokens: tokens,
		},
		Model: c.options.model.APIModel,
	}, nil
}
