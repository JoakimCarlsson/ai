package llm

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

type Embedding interface {
	GenerateEmbeddings(ctx context.Context, texts []string) (*EmbeddingResponse, error)
	GenerateMultimodalEmbeddings(ctx context.Context, inputs []MultimodalInput) (*EmbeddingResponse, error)
	Model() model.EmbeddingModel
}

type embeddingClientOptions struct {
	apiKey    string
	model     model.EmbeddingModel
	batchSize int
	timeout   *time.Duration

	voyageOptions []VoyageOption
}

type EmbeddingClientOption func(*embeddingClientOptions)

type EmbeddingClient interface {
	embed(ctx context.Context, texts []string) (*EmbeddingResponse, error)
	embedMultimodal(ctx context.Context, inputs []MultimodalInput) (*EmbeddingResponse, error)
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
	}

	return nil, fmt.Errorf("embedding provider not supported: %s", provider)
}

func (e *baseEmbedding[C]) GenerateEmbeddings(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embed(ctx, texts)
}

func (e *baseEmbedding[C]) GenerateMultimodalEmbeddings(ctx context.Context, inputs []MultimodalInput) (*EmbeddingResponse, error) {
	if len(inputs) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embedMultimodal(ctx, inputs)
}

func (e *baseEmbedding[C]) Model() model.EmbeddingModel {
	return e.options.model
}

func WithEmbeddingAPIKey(apiKey string) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.apiKey = apiKey
	}
}

func WithEmbeddingModel(model model.EmbeddingModel) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.model = model
	}
}

func WithBatchSize(batchSize int) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.batchSize = batchSize
	}
}

func WithEmbeddingTimeout(timeout time.Duration) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.timeout = &timeout
	}
}

func WithVoyageOptions(voyageOptions ...VoyageOption) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.voyageOptions = voyageOptions
	}
}
