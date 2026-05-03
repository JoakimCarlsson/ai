// Package mistral provides a Mistral implementation of the [embeddings.Embedding] interface.
package mistral

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

const defaultBaseURL = "https://api.mistral.ai/v1"

// Options configures the Mistral embeddings client.
type Options struct {
	apiKey          string
	model           model.EmbeddingModel
	timeout         *time.Duration
	batchSize       int
	dimensions      *int
	outputDimension *int
	outputDtype     string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Mistral.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the embedding model.
func WithModel(m model.EmbeddingModel) Option { return func(o *Options) { o.model = m } }

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithBatchSize sets the number of texts to process in each batch request.
func WithBatchSize(batchSize int) Option { return func(o *Options) { o.batchSize = batchSize } }

// WithDimensions specifies the output dimensionality for embedding vectors.
// Equivalent to [WithOutputDimension]; Mistral accepts either.
func WithDimensions(dimensions int) Option { return func(o *Options) { o.dimensions = &dimensions } }

// WithOutputDimension sets the output embedding dimensionality (codestral-embed only).
func WithOutputDimension(dim int) Option {
	return func(o *Options) { o.outputDimension = &dim }
}

// WithOutputDtype sets the output data type ("float", "int8", "uint8", "binary", "ubinary").
func WithOutputDtype(dtype string) Option { return func(o *Options) { o.outputDtype = dtype } }

// Client implements [embeddings.Embedding] against the Mistral API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewEmbedding constructs a Mistral embeddings client.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	options := Options{batchSize: 512}
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
	})
}

// Model returns the configured embedding model.
func (c *Client) Model() model.EmbeddingModel { return c.options.model }

type embedRequest struct {
	Model           string   `json:"model"`
	Input           []string `json:"input"`
	OutputDimension *int     `json:"output_dimension,omitempty"`
	OutputDtype     string   `json:"output_dtype,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int64 `json:"prompt_tokens"`
		TotalTokens  int64 `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateEmbeddings creates vector embeddings from text strings.
func (c *Client) GenerateEmbeddings(
	ctx context.Context,
	texts []string,
	_ ...string,
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
		batchSize = 512
	}

	var allEmbeddings [][]float32
	var totalTokens int64

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		resp, err := c.embedBatch(ctx, texts[i:end])
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
) (*embeddings.EmbeddingResponse, error) {
	reqBody := embedRequest{
		Model: c.options.model.APIModel,
		Input: texts,
	}

	if c.options.dimensions != nil {
		reqBody.OutputDimension = c.options.dimensions
	} else if c.options.outputDimension != nil {
		reqBody.OutputDimension = c.options.outputDimension
	}
	if c.options.outputDtype != "" {
		reqBody.OutputDtype = c.options.outputDtype
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", c.baseURL+"/embeddings", bytes.NewBuffer(jsonBody),
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
		return nil, fmt.Errorf("failed to read embed response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var mResp embedResponse
	if err := json.Unmarshal(body, &mResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embed response: %w", err)
	}

	out := make([][]float32, len(mResp.Data))
	for _, d := range mResp.Data {
		out[d.Index] = d.Embedding
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: out,
		Usage:      embeddings.EmbeddingUsage{TotalTokens: mResp.Usage.TotalTokens},
		Model:      mResp.Model,
	}, nil
}

// GenerateMultimodalEmbeddings is not supported by Mistral.
func (c *Client) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []embeddings.MultimodalInput,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	return nil, fmt.Errorf("mistral does not support multimodal embeddings")
}

// GenerateContextualizedEmbeddings is not supported by Mistral.
func (c *Client) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf("mistral does not support contextualized embeddings")
}
