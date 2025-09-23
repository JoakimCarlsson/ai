package embeddings

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

type EmbeddingUsage struct {
	TotalTokens int64
	TextTokens  int64
	ImagePixels int64
}

type MultimodalContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
}

type MultimodalInput struct {
	Content []MultimodalContent `json:"content"`
}

type EmbeddingResponse struct {
	Embeddings [][]float32
	Usage      EmbeddingUsage
	Model      string
}

type ContextualizedEmbeddingResponse struct {
	DocumentEmbeddings [][][]float32
	Usage              EmbeddingUsage
	Model              string
}

type Embedding interface {
	GenerateEmbeddings(ctx context.Context, texts []string, inputType ...string) (*EmbeddingResponse, error)
	GenerateMultimodalEmbeddings(ctx context.Context, inputs []MultimodalInput, inputType ...string) (*EmbeddingResponse, error)
	GenerateContextualizedEmbeddings(ctx context.Context, documentChunks [][]string, inputType ...string) (*ContextualizedEmbeddingResponse, error)
	Model() model.EmbeddingModel
}

type embeddingClientOptions struct {
	apiKey     string
	model      model.EmbeddingModel
	batchSize  int
	timeout    *time.Duration
	dimensions *int

	voyageOptions []VoyageOption
	openaiOptions []OpenAIOption
}

type EmbeddingClientOption func(*embeddingClientOptions)

type EmbeddingClient interface {
	embed(ctx context.Context, texts []string, inputType ...string) (*EmbeddingResponse, error)
	embedMultimodal(ctx context.Context, inputs []MultimodalInput, inputType ...string) (*EmbeddingResponse, error)
	embedContextualized(ctx context.Context, documentChunks [][]string, inputType ...string) (*ContextualizedEmbeddingResponse, error)
}

type baseEmbedding[C EmbeddingClient] struct {
	options embeddingClientOptions
	client  C
}

func NewEmbedding(provider model.ModelProvider, opts ...EmbeddingClientOption) (Embedding, error) {
	clientOptions := embeddingClientOptions{
		batchSize: 100,
	}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderVoyage:
		return &baseEmbedding[VoyageClient]{
			options: clientOptions,
			client:  newVoyageClient(clientOptions),
		}, nil
	case model.ProviderOpenAI:
		return &baseEmbedding[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf("embedding provider not supported: %s", provider)
}

func (e *baseEmbedding[C]) GenerateEmbeddings(ctx context.Context, texts []string, inputType ...string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embed(ctx, texts, inputType...)
}

func (e *baseEmbedding[C]) GenerateMultimodalEmbeddings(ctx context.Context, inputs []MultimodalInput, inputType ...string) (*EmbeddingResponse, error) {
	if len(inputs) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embedMultimodal(ctx, inputs, inputType...)
}

func (e *baseEmbedding[C]) GenerateContextualizedEmbeddings(ctx context.Context, documentChunks [][]string, inputType ...string) (*ContextualizedEmbeddingResponse, error) {
	if len(documentChunks) == 0 {
		return &ContextualizedEmbeddingResponse{
			DocumentEmbeddings: [][][]float32{},
			Usage:              EmbeddingUsage{TotalTokens: 0},
			Model:              e.options.model.APIModel,
		}, nil
	}

	return e.client.embedContextualized(ctx, documentChunks, inputType...)
}

func (e *baseEmbedding[C]) Model() model.EmbeddingModel {
	return e.options.model
}

func WithAPIKey(apiKey string) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.apiKey = apiKey
	}
}

func WithModel(model model.EmbeddingModel) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.model = model
	}
}

func WithBatchSize(batchSize int) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.batchSize = batchSize
	}
}

func WithTimeout(timeout time.Duration) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.timeout = &timeout
	}
}

func WithDimensions(dimensions int) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.dimensions = &dimensions
	}
}

func WithVoyageOptions(voyageOptions ...VoyageOption) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.voyageOptions = voyageOptions
	}
}

func WithOpenAIOptions(openaiOptions ...OpenAIOption) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.openaiOptions = openaiOptions
	}
}
