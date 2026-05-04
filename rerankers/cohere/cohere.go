// Package cohere provides a Cohere implementation of the [rerankers.Reranker] interface.
package cohere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rerankers"
)

const defaultBaseURL = "https://api.cohere.com/v2"

// Options configures the Cohere reranker client.
type Options struct {
	apiKey          string
	model           model.RerankerModel
	timeout         *time.Duration
	topK            *int
	returnDocs      bool
	maxChunksPerDoc *int
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Cohere.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) {
		o.apiKey = apiKey
	}
}

// WithModel selects the reranker model.
func WithModel(m model.RerankerModel) Option {
	return func(o *Options) {
		o.model = m
	}
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.timeout = &timeout
	}
}

// WithTopK limits how many top-ranked documents are returned.
func WithTopK(topK int) Option {
	return func(o *Options) {
		o.topK = &topK
	}
}

// WithReturnDocuments controls whether document text is included in results.
func WithReturnDocuments(returnDocs bool) Option {
	return func(o *Options) {
		o.returnDocs = returnDocs
	}
}

// WithMaxChunksPerDoc sets the maximum number of chunks per document for reranking.
func WithMaxChunksPerDoc(maxChunks int) Option {
	return func(o *Options) {
		o.maxChunksPerDoc = &maxChunks
	}
}

// Client implements [rerankers.Reranker] against the Cohere reranker API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewReranker constructs a Cohere reranker client. The returned [rerankers.Reranker]
// is wrapped with [rerankers.WithTracing], so callers always get tracing spans and metrics.
func NewReranker(opts ...Option) rerankers.Reranker {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return rerankers.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
	}, rerankers.TracingAttrs{
		TopK:            options.topK,
		MaxChunksPerDoc: options.maxChunksPerDoc,
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
	MaxChunksPerDoc *int     `json:"max_chunks_per_doc,omitempty"`
}

type response struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
		Document       *struct {
			Text string `json:"text"`
		} `json:"document,omitempty"`
	} `json:"results"`
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
		MaxChunksPerDoc: c.options.maxChunksPerDoc,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reranker request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/rerank",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reranker request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make reranker request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read reranker response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"reranker API request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var cohereResp response
	if err := json.Unmarshal(body, &cohereResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reranker response: %w", err)
	}

	results := make([]rerankers.RerankerResult, len(cohereResp.Results))
	for i, data := range cohereResp.Results {
		result := rerankers.RerankerResult{
			Index:          data.Index,
			RelevanceScore: data.RelevanceScore,
		}
		if data.Document != nil {
			result.Document = data.Document.Text
		}
		results[i] = result
	}

	return &rerankers.RerankerResponse{
		Results: results,
		Usage:   rerankers.RerankerUsage{TotalTokens: cohereResp.Meta.BilledUnits.SearchUnits},
		Model:   c.options.model.APIModel,
	}, nil
}
