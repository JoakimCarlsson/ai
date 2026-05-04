// Package gemini provides a Google Gemini implementation of the [embeddings.Embedding] interface.
package gemini

import (
	"context"
	"fmt"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
	"google.golang.org/genai"
)

// Options configures the Gemini embeddings client.
type Options struct {
	apiKey     string
	model      model.EmbeddingModel
	batchSize  int
	dimensions *int
	taskType   string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Gemini.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the embedding model.
func WithModel(m model.EmbeddingModel) Option { return func(o *Options) { o.model = m } }

// WithBatchSize sets the number of texts to process in each batch request.
func WithBatchSize(batchSize int) Option { return func(o *Options) { o.batchSize = batchSize } }

// WithDimensions specifies the output dimensionality for embedding vectors.
func WithDimensions(dimensions int) Option { return func(o *Options) { o.dimensions = &dimensions } }

// WithTaskType sets the task type for embeddings (e.g., "RETRIEVAL_DOCUMENT", "RETRIEVAL_QUERY").
func WithTaskType(taskType string) Option { return func(o *Options) { o.taskType = taskType } }

// Client implements [embeddings.Embedding] against the Google Gemini API.
type Client struct {
	options Options
	client  *genai.Client
}

// NewEmbedding constructs a Gemini embeddings client.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	options := Options{batchSize: 100}
	for _, o := range opts {
		o(&options)
	}

	client, _ := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey:  options.apiKey,
			Backend: genai.BackendGeminiAPI,
		},
	)

	return embeddings.WithTracing(&Client{
		options: options,
		client:  client,
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
		batchSize = 100
	}

	var allEmbeddings [][]float32

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
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      embeddings.EmbeddingUsage{},
		Model:      c.options.model.APIModel,
	}, nil
}

func (c *Client) embedBatch(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = genai.NewContentFromText(text, "user")
	}

	config := &genai.EmbedContentConfig{}
	taskType := c.options.taskType
	if len(inputType) > 0 && inputType[0] != "" {
		taskType = inputType[0]
	}
	if taskType != "" {
		config.TaskType = taskType
	}
	if c.options.dimensions != nil {
		dim := int32(*c.options.dimensions)
		config.OutputDimensionality = &dim
	}

	modelName := c.options.model.APIModel
	resp, err := c.client.Models.EmbedContent(ctx, modelName, contents, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	out := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		out[i] = emb.Values
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: out,
		Usage:      embeddings.EmbeddingUsage{},
		Model:      modelName,
	}, nil
}

// GenerateMultimodalEmbeddings is not supported by Gemini.
func (c *Client) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []embeddings.MultimodalInput,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	return nil, fmt.Errorf("gemini does not support multimodal embeddings")
}

// GenerateContextualizedEmbeddings is not supported by Gemini.
func (c *Client) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf("gemini does not support contextualized embeddings")
}
