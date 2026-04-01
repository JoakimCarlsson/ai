package rerankers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type cohereOptions struct {
	maxChunksPerDoc *int
}

// CohereOption configures Cohere-specific reranker behavior.
type CohereOption func(*cohereOptions)

type cohereClient struct {
	providerOptions rerankerClientOptions
	options         cohereOptions
	httpClient      *http.Client
	baseURL         string
}

// CohereClient is the Cohere implementation of RerankerClient.
type CohereClient RerankerClient

type cohereRerankerRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            *int     `json:"top_n,omitempty"`
	ReturnDocuments bool     `json:"return_documents,omitempty"`
	MaxChunksPerDoc *int     `json:"max_chunks_per_doc,omitempty"`
}

type cohereRerankerResponse struct {
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

func newCohereClient(
	opts rerankerClientOptions,
) CohereClient {
	cohereOpts := cohereOptions{}
	for _, o := range opts.cohereOptions {
		o(&cohereOpts)
	}

	timeout := 30 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &cohereClient{
		providerOptions: opts,
		options:         cohereOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.cohere.com/v2",
	}
}

func (c *cohereClient) rerank(
	ctx context.Context,
	query string,
	documents []string,
) (*RerankerResponse, error) {
	if len(documents) == 0 {
		return &RerankerResponse{
			Results: []RerankerResult{},
			Usage:   RerankerUsage{TotalTokens: 0},
			Model:   c.providerOptions.model.APIModel,
		}, nil
	}

	reqBody := cohereRerankerRequest{
		Model:           c.providerOptions.model.APIModel,
		Query:           query,
		Documents:       documents,
		TopN:            c.providerOptions.topK,
		ReturnDocuments: c.providerOptions.returnDocs,
		MaxChunksPerDoc: c.options.maxChunksPerDoc,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal reranker request: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/rerank",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create reranker request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(
		"Authorization",
		"Bearer "+c.providerOptions.apiKey,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make reranker request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read reranker response body: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"reranker API request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var cohereResp cohereRerankerResponse
	if err := json.Unmarshal(body, &cohereResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal reranker response: %w",
			err,
		)
	}

	results := make(
		[]RerankerResult,
		len(cohereResp.Results),
	)
	for i, data := range cohereResp.Results {
		result := RerankerResult{
			Index:          data.Index,
			RelevanceScore: data.RelevanceScore,
		}
		if data.Document != nil {
			result.Document = data.Document.Text
		}
		results[i] = result
	}

	return &RerankerResponse{
		Results: results,
		Usage: RerankerUsage{
			TotalTokens: cohereResp.Meta.BilledUnits.SearchUnits,
		},
		Model: c.providerOptions.model.APIModel,
	}, nil
}

// WithCohereMaxChunksPerDoc sets the maximum number of chunks per document for reranking.
func WithCohereMaxChunksPerDoc(
	maxChunks int,
) CohereOption {
	return func(options *cohereOptions) {
		options.maxChunksPerDoc = &maxChunks
	}
}
