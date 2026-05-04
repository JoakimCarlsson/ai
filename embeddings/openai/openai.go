// Package openai provides an OpenAI implementation of the [embeddings.Embedding] interface.
package openai

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Options configures the OpenAI embeddings client.
type Options struct {
	apiKey     string
	model      model.EmbeddingModel
	timeout    *time.Duration
	batchSize  int
	dimensions *int
	baseURL    string
	user       string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with OpenAI.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the embedding model.
func WithModel(m model.EmbeddingModel) Option { return func(o *Options) { o.model = m } }

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithBatchSize sets the number of texts to process in each batch request.
func WithBatchSize(batchSize int) Option { return func(o *Options) { o.batchSize = batchSize } }

// WithDimensions specifies the output dimensionality for embedding vectors.
func WithDimensions(dimensions int) Option { return func(o *Options) { o.dimensions = &dimensions } }

// WithBaseURL points the client at a custom OpenAI-compatible endpoint.
func WithBaseURL(baseURL string) Option { return func(o *Options) { o.baseURL = baseURL } }

// WithUser sets a unique identifier for the end-user (helps OpenAI monitor/detect abuse).
func WithUser(user string) Option { return func(o *Options) { o.user = user } }

// Client implements [embeddings.Embedding] against the OpenAI embeddings API.
type Client struct {
	options Options
	client  openaisdk.Client
}

// NewEmbedding constructs an OpenAI embeddings client. The returned [embeddings.Embedding]
// is wrapped with [embeddings.WithTracing], so callers always get tracing spans and metrics.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	options := Options{batchSize: 100}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{}
	if options.apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(options.apiKey))
	}
	if options.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(options.baseURL))
	}

	return embeddings.WithTracing(&Client{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, embeddings.TracingAttrs{
		Dimensions: options.dimensions,
	})
}

// Model returns the configured embedding model.
func (c *Client) Model() model.EmbeddingModel { return c.options.model }

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
		batchSize = 2048
	}

	var allEmbeddings [][]float32
	var totalTokens int64

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		response, err := c.embedBatch(ctx, texts[i:end])
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch: %w", err)
		}

		allEmbeddings = append(allEmbeddings, response.Embeddings...)
		totalTokens += response.Usage.TotalTokens
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
	params := openaisdk.EmbeddingNewParams{
		Model: openaisdk.EmbeddingModel(c.options.model.APIModel),
		Input: openaisdk.EmbeddingNewParamsInputUnion{OfArrayOfStrings: texts},
	}

	if c.options.dimensions != nil {
		params.Dimensions = openaisdk.Int(int64(*c.options.dimensions))
	}
	if c.options.user != "" {
		params.User = openaisdk.String(c.options.user)
	}

	resp, err := c.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	out := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embedding := make([]float32, len(data.Embedding))
		for j, val := range data.Embedding {
			embedding[j] = float32(val)
		}
		out[i] = embedding
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: out,
		Usage:      embeddings.EmbeddingUsage{TotalTokens: int64(resp.Usage.TotalTokens)},
		Model:      string(resp.Model),
	}, nil
}

// GenerateMultimodalEmbeddings is not supported by OpenAI.
func (c *Client) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []embeddings.MultimodalInput,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	return nil, fmt.Errorf("OpenAI does not support multimodal embeddings")
}

// GenerateContextualizedEmbeddings is not supported by OpenAI.
func (c *Client) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf("OpenAI does not support contextualized embeddings")
}
