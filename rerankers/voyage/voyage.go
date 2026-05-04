// Package voyage provides a Voyage AI implementation of the [rerankers.Reranker] interface.
package voyage

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

const defaultBaseURL = "https://api.voyageai.com/v1"

// Options configures the Voyage reranker client.
type Options struct {
	apiKey     string
	model      model.RerankerModel
	timeout    *time.Duration
	topK       *int
	returnDocs bool
	truncation *bool
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Voyage AI.
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

// WithTruncation enables or disables input truncation to fit the model context.
func WithTruncation(truncation bool) Option {
	return func(o *Options) {
		o.truncation = &truncation
	}
}

// Client implements [rerankers.Reranker] against the Voyage AI reranker API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewReranker constructs a Voyage reranker client. The returned [rerankers.Reranker]
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
		ReturnDocuments: &options.returnDocs,
	})
}

// Model returns the configured reranker model.
func (c *Client) Model() model.RerankerModel {
	return c.options.model
}

type request struct {
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	Model           string   `json:"model"`
	TopK            *int     `json:"top_k,omitempty"`
	ReturnDocuments bool     `json:"return_documents,omitempty"`
	Truncation      *bool    `json:"truncation,omitempty"`
}

type response struct {
	Object string `json:"object"`
	Data   []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
		Document       string  `json:"document,omitempty"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int64 `json:"total_tokens"`
	} `json:"usage"`
}

// Rerank reorders documents by relevance to the query.
func (c *Client) Rerank(
	ctx context.Context,
	query string,
	documents []string,
) (*rerankers.RerankerResponse, error) {
	reqBody := request{
		Query:           query,
		Documents:       documents,
		Model:           c.options.model.APIModel,
		TopK:            c.options.topK,
		ReturnDocuments: c.options.returnDocs,
		Truncation:      c.options.truncation,
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

	var voyageResp response
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reranker response: %w", err)
	}

	results := make([]rerankers.RerankerResult, len(voyageResp.Data))
	for i, data := range voyageResp.Data {
		results[i] = rerankers.RerankerResult{
			Index:          data.Index,
			RelevanceScore: data.RelevanceScore,
			Document:       data.Document,
		}
	}

	return &rerankers.RerankerResponse{
		Results: results,
		Usage:   rerankers.RerankerUsage{TotalTokens: voyageResp.Usage.TotalTokens},
		Model:   voyageResp.Model,
	}, nil
}
