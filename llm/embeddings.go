package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// EmbeddingUsage tracks token usage for an embedding request.
type EmbeddingUsage struct {
	TotalTokens int64
	TextTokens  int64
	ImagePixels int64
}

// MultimodalContent represents a single content element in a multimodal embedding input.
type MultimodalContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
}

// MultimodalInput groups multiple content elements for a single multimodal embedding.
type MultimodalInput struct {
	Content []MultimodalContent `json:"content"`
}

// EmbeddingResponse contains the generated embeddings and usage metadata.
type EmbeddingResponse struct {
	Embeddings [][]float32
	Usage      EmbeddingUsage
	Model      string
}

// ContextualizedEmbeddingResponse contains document-level contextualized embeddings.
type ContextualizedEmbeddingResponse struct {
	DocumentEmbeddings [][][]float32
	Usage              EmbeddingUsage
	Model              string
}

// Embedding defines the interface for generating text and multimodal embeddings.
type Embedding interface {
	GenerateEmbeddings(
		ctx context.Context,
		texts []string,
	) (*EmbeddingResponse, error)
	GenerateMultimodalEmbeddings(
		ctx context.Context,
		inputs []MultimodalInput,
	) (*EmbeddingResponse, error)
	GenerateContextualizedEmbeddings(
		ctx context.Context,
		documentChunks [][]string,
	) (*ContextualizedEmbeddingResponse, error)
	Model() model.EmbeddingModel
}

type embeddingClientOptions struct {
	apiKey    string
	model     model.EmbeddingModel
	batchSize int
	timeout   *time.Duration

	voyageOptions []VoyageOption
}

// EmbeddingClientOption configures an embedding client when passed to NewEmbedding.
type EmbeddingClientOption func(*embeddingClientOptions)

// EmbeddingClient defines the provider-specific implementation for embedding generation.
type EmbeddingClient interface {
	embed(ctx context.Context, texts []string) (*EmbeddingResponse, error)
	embedMultimodal(
		ctx context.Context,
		inputs []MultimodalInput,
	) (*EmbeddingResponse, error)
	embedContextualized(
		ctx context.Context,
		documentChunks [][]string,
	) (*ContextualizedEmbeddingResponse, error)
}

type baseEmbedding[C EmbeddingClient] struct {
	options embeddingClientOptions
	client  C
}

// NewEmbedding creates a new embedding client for the specified provider.
func NewEmbedding(
	provider model.Provider,
	opts ...EmbeddingClientOption,
) (Embedding, error) {
	clientOptions := embeddingClientOptions{
		batchSize: 100,
	}
	for _, o := range opts {
		o(&clientOptions)
	}

	if provider == model.ProviderVoyage {
		return &baseEmbedding[VoyageClient]{
			options: clientOptions,
			client:  newVoyageClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf("embedding provider not supported: %s", provider)
}

func (e *baseEmbedding[C]) GenerateEmbeddings(
	ctx context.Context,
	texts []string,
) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embed(ctx, texts)
}

func (e *baseEmbedding[C]) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []MultimodalInput,
) (*EmbeddingResponse, error) {
	if len(inputs) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embedMultimodal(ctx, inputs)
}

func (e *baseEmbedding[C]) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
) (*ContextualizedEmbeddingResponse, error) {
	if len(documentChunks) == 0 {
		return &ContextualizedEmbeddingResponse{
			DocumentEmbeddings: [][][]float32{},
			Usage:              EmbeddingUsage{TotalTokens: 0},
			Model:              e.options.model.APIModel,
		}, nil
	}

	return e.client.embedContextualized(ctx, documentChunks)
}

func (e *baseEmbedding[C]) Model() model.EmbeddingModel {
	return e.options.model
}

// WithEmbeddingAPIKey sets the API key for the embedding provider.
func WithEmbeddingAPIKey(apiKey string) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.apiKey = apiKey
	}
}

// WithEmbeddingModel specifies which embedding model to use.
func WithEmbeddingModel(model model.EmbeddingModel) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.model = model
	}
}

// WithBatchSize sets the number of texts to embed per API request.
func WithBatchSize(batchSize int) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.batchSize = batchSize
	}
}

// WithEmbeddingTimeout sets the maximum duration for embedding API requests.
func WithEmbeddingTimeout(timeout time.Duration) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.timeout = &timeout
	}
}

// WithVoyageOptions applies Voyage-specific configuration options.
func WithVoyageOptions(voyageOptions ...VoyageOption) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.voyageOptions = voyageOptions
	}
}
