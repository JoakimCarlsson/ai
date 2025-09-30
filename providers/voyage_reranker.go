package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type voyageRerankerOptions struct {
	topK       *int
	returnDocs bool
	truncation *bool
}

type VoyageRerankerOption func(*voyageRerankerOptions)

type voyageRerankerClient struct {
	providerOptions rerankerClientOptions
	options         voyageRerankerOptions
	httpClient      *http.Client
	baseURL         string
}

type VoyageRerankerClient RerankerClient

type voyageRerankerRequest struct {
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	Model           string   `json:"model"`
	TopK            *int     `json:"top_k,omitempty"`
	ReturnDocuments bool     `json:"return_documents,omitempty"`
	Truncation      *bool    `json:"truncation,omitempty"`
}

type voyageRerankerResponse struct {
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

func newVoyageRerankerClient(opts rerankerClientOptions) VoyageRerankerClient {
	voyageOpts := voyageRerankerOptions{
		returnDocs: opts.returnDocs,
		truncation: opts.truncation,
		topK:       opts.topK,
	}
	for _, o := range opts.voyageOptions {
		o(&voyageOpts)
	}

	timeout := 30 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &voyageRerankerClient{
		providerOptions: opts,
		options:         voyageOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.voyageai.com/v1",
	}
}

func (v *voyageRerankerClient) rerank(
	ctx context.Context,
	query string,
	documents []string,
) (*RerankerResponse, error) {
	if len(documents) == 0 {
		return &RerankerResponse{
			Results: []RerankerResult{},
			Usage:   RerankerUsage{TotalTokens: 0},
			Model:   v.providerOptions.model.APIModel,
		}, nil
	}

	reqBody := voyageRerankerRequest{
		Query:           query,
		Documents:       documents,
		Model:           v.providerOptions.model.APIModel,
		TopK:            v.options.topK,
		ReturnDocuments: v.options.returnDocs,
		Truncation:      v.options.truncation,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reranker request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		v.baseURL+"/rerank",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reranker request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.providerOptions.apiKey)

	resp, err := v.httpClient.Do(req)
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

	var voyageResp voyageRerankerResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reranker response: %w", err)
	}

	results := make([]RerankerResult, len(voyageResp.Data))
	for i, data := range voyageResp.Data {
		results[i] = RerankerResult{
			Index:          data.Index,
			RelevanceScore: data.RelevanceScore,
			Document:       data.Document,
		}
	}

	return &RerankerResponse{
		Results: results,
		Usage:   RerankerUsage{TotalTokens: voyageResp.Usage.TotalTokens},
		Model:   voyageResp.Model,
	}, nil
}

func WithVoyageTopK(topK int) VoyageRerankerOption {
	return func(options *voyageRerankerOptions) {
		options.topK = &topK
	}
}

func WithVoyageReturnDocuments(returnDocs bool) VoyageRerankerOption {
	return func(options *voyageRerankerOptions) {
		options.returnDocs = returnDocs
	}
}

func WithVoyageRerankerTruncation(truncation bool) VoyageRerankerOption {
	return func(options *voyageRerankerOptions) {
		options.truncation = &truncation
	}
}
