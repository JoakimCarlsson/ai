package embeddings

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
	inputType      string
	truncation     string
	embeddingTypes []string
}

// CohereOption configures Cohere-specific embedding behavior.
type CohereOption func(*cohereOptions)

type cohereClient struct {
	providerOptions embeddingClientOptions
	options         cohereOptions
	httpClient      *http.Client
	baseURL         string
}

// CohereClient is the Cohere implementation of EmbeddingClient.
type CohereClient EmbeddingClient

type cohereEmbedRequest struct {
	Model          string   `json:"model"`
	Texts          []string `json:"texts"`
	InputType      string   `json:"input_type,omitempty"`
	Truncate       string   `json:"truncate,omitempty"`
	EmbeddingTypes []string `json:"embedding_types,omitempty"`
}

type cohereEmbedResponse struct {
	Embeddings struct {
		Float [][]float32 `json:"float"`
	} `json:"embeddings"`
	Meta struct {
		BilledUnits struct {
			InputTokens int64 `json:"input_tokens"`
		} `json:"billed_units"`
	} `json:"meta"`
}

func newCohereClient(
	opts embeddingClientOptions,
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

func (c *cohereClient) embed(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      c.providerOptions.model.APIModel,
		}, nil
	}

	batchSize := c.providerOptions.batchSize
	if batchSize <= 0 {
		batchSize = 96
	}

	var allEmbeddings [][]float32
	var totalTokens int64

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		resp, err := c.embedBatch(
			ctx,
			texts[i:end],
			inputType...,
		)
		if err != nil {
			return nil, err
		}

		allEmbeddings = append(
			allEmbeddings,
			resp.Embeddings...,
		)
		totalTokens += resp.Usage.TotalTokens
	}

	return &EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      EmbeddingUsage{TotalTokens: totalTokens},
		Model:      c.providerOptions.model.APIModel,
	}, nil
}

func (c *cohereClient) embedBatch(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	reqBody := cohereEmbedRequest{
		Model:          c.providerOptions.model.APIModel,
		Texts:          texts,
		InputType:      c.options.inputType,
		Truncate:       c.options.truncation,
		EmbeddingTypes: c.options.embeddingTypes,
	}

	if len(inputType) > 0 && inputType[0] != "" {
		reqBody.InputType = inputType[0]
	}

	if len(reqBody.EmbeddingTypes) == 0 {
		reqBody.EmbeddingTypes = []string{"float"}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal embed request: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/embed",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create embed request: %w",
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
			"failed to make embed request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read embed response body: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"embed API request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var cohereResp cohereEmbedResponse
	if err := json.Unmarshal(body, &cohereResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal embed response: %w",
			err,
		)
	}

	return &EmbeddingResponse{
		Embeddings: cohereResp.Embeddings.Float,
		Usage: EmbeddingUsage{
			TotalTokens: cohereResp.Meta.BilledUnits.InputTokens,
		},
		Model: c.providerOptions.model.APIModel,
	}, nil
}

func (c *cohereClient) embedMultimodal(
	_ context.Context,
	_ []MultimodalInput,
	_ ...string,
) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"cohere does not support multimodal embeddings",
	)
}

func (c *cohereClient) embedContextualized(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"cohere does not support contextualized embeddings",
	)
}

// WithCohereInputType sets the input type for embeddings (e.g., "search_document", "search_query").
func WithCohereInputType(inputType string) CohereOption {
	return func(options *cohereOptions) {
		options.inputType = inputType
	}
}

// WithCohereTruncation sets the truncation strategy (e.g., "NONE", "START", "END").
func WithCohereTruncation(truncation string) CohereOption {
	return func(options *cohereOptions) {
		options.truncation = truncation
	}
}

// WithCohereEmbeddingTypes sets the embedding types to return (e.g., "float", "int8", "uint8", "binary", "ubinary").
func WithCohereEmbeddingTypes(
	types []string,
) CohereOption {
	return func(options *cohereOptions) {
		options.embeddingTypes = types
	}
}
