// Package cohere provides a Cohere implementation of the [embeddings.Embedding] interface.
package cohere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

const defaultBaseURL = "https://api.cohere.com/v2"

// Options configures the Cohere embeddings client.
type Options struct {
	apiKey         string
	model          model.EmbeddingModel
	timeout        *time.Duration
	batchSize      int
	inputType      string
	truncation     string
	embeddingTypes []string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Cohere.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the embedding model.
func WithModel(
	m model.EmbeddingModel,
) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithBatchSize sets the number of texts to process in each batch request.
func WithBatchSize(
	batchSize int,
) Option {
	return func(o *Options) { o.batchSize = batchSize }
}

// WithInputType sets the input type for embeddings (e.g., "search_document", "search_query").
func WithInputType(
	inputType string,
) Option {
	return func(o *Options) { o.inputType = inputType }
}

// WithTruncation sets the truncation strategy (e.g., "NONE", "START", "END").
func WithTruncation(truncation string) Option {
	return func(o *Options) { o.truncation = truncation }
}

// WithEmbeddingTypes sets the embedding types to return (e.g., "float", "int8", "binary").
func WithEmbeddingTypes(types []string) Option {
	return func(o *Options) { o.embeddingTypes = types }
}

// Client implements [embeddings.Embedding] against the Cohere API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewEmbedding constructs a Cohere embeddings client.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	options := Options{batchSize: 96}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return embeddings.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
	}, embeddings.TracingAttrs{})
}

// Model returns the configured embedding model.
func (c *Client) Model() model.EmbeddingModel { return c.options.model }

type embedRequest struct {
	Model          string   `json:"model"`
	Texts          []string `json:"texts"`
	InputType      string   `json:"input_type,omitempty"`
	Truncate       string   `json:"truncate,omitempty"`
	EmbeddingTypes []string `json:"embedding_types,omitempty"`
}

type embedResponse struct {
	Embeddings struct {
		Float [][]float32 `json:"float"`
	} `json:"embeddings"`
	Meta struct {
		BilledUnits struct {
			InputTokens int64 `json:"input_tokens"`
		} `json:"billed_units"`
	} `json:"meta"`
}

// GenerateEmbeddings creates vector embeddings from text strings.
func (c *Client) GenerateEmbeddings(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &embeddings.EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      embeddings.EmbeddingUsage{TotalTokens: 0},
			Model:      c.options.model.APIModel,
		}, nil
	}

	batchSize := c.options.batchSize
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

		resp, err := c.embedBatch(ctx, texts[i:end], inputType...)
		if err != nil {
			return nil, err
		}

		allEmbeddings = append(allEmbeddings, resp.Embeddings...)
		totalTokens += resp.Usage.TotalTokens
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      embeddings.EmbeddingUsage{TotalTokens: totalTokens},
		Model:      c.options.model.APIModel,
	}, nil
}

func (c *Client) embedBatch(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	reqBody := embedRequest{
		Model:          c.options.model.APIModel,
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
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", c.baseURL+"/embed", bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make embed request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embed response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"embed API request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var cohereResp embedResponse
	if err := json.Unmarshal(body, &cohereResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embed response: %w", err)
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: cohereResp.Embeddings.Float,
		Usage: embeddings.EmbeddingUsage{
			TotalTokens: cohereResp.Meta.BilledUnits.InputTokens,
		},
		Model: c.options.model.APIModel,
	}, nil
}

// GenerateMultimodalEmbeddings is not supported by Cohere.
func (c *Client) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []embeddings.MultimodalInput,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	return nil, fmt.Errorf("cohere does not support multimodal embeddings")
}

// GenerateContextualizedEmbeddings is not supported by Cohere.
func (c *Client) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf("cohere does not support contextualized embeddings")
}
