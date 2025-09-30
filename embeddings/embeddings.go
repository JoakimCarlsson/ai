// Package embeddings provides a unified interface for generating text and multimodal embeddings
// from various AI providers.
//
// This package abstracts the differences between embedding providers like Voyage AI and OpenAI,
// offering a consistent API for generating vector embeddings from text, images, and mixed content.
// It supports standard embeddings, multimodal embeddings, and contextualized embeddings for
// improved document understanding.
//
// Key features include:
//   - Text embedding generation from strings
//   - Multimodal embedding generation from text and images
//   - Contextualized embeddings for better document chunk understanding
//   - Automatic batching for efficient processing
//   - Token usage tracking and cost calculation
//   - Provider-specific optimizations and features
//
// Example usage:
//
//	embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
//		embeddings.WithAPIKey("your-api-key"),
//		embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	texts := []string{"Hello world", "How are you?"}
//	response, err := embedder.GenerateEmbeddings(ctx, texts)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Generated %d embeddings\n", len(response.Embeddings))
package embeddings

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// EmbeddingUsage tracks the resource consumption for embedding generation.
type EmbeddingUsage struct {
	// TotalTokens is the total number of tokens processed.
	TotalTokens int64
	// TextTokens is the number of text tokens processed.
	TextTokens int64
	// ImagePixels is the number of image pixels processed for multimodal embeddings.
	ImagePixels int64
}

// MultimodalContent represents a single piece of content that can be text or image.
type MultimodalContent struct {
	// Type indicates the content type ("text" or "image_url" or "image_base64").
	Type string `json:"type"`
	// Text contains the text content when Type is "text".
	Text string `json:"text,omitempty"`
	// ImageURL contains the URL to an image when Type is "image_url".
	ImageURL string `json:"image_url,omitempty"`
	// ImageBase64 contains base64-encoded image data when Type is "image_base64".
	ImageBase64 string `json:"image_base64,omitempty"`
}

// MultimodalInput represents a collection of multimodal content pieces for embedding generation.
type MultimodalInput struct {
	// Content contains the mixed text and image content to embed.
	Content []MultimodalContent `json:"content"`
}

// EmbeddingResponse contains the generated embeddings and metadata from an embedding request.
type EmbeddingResponse struct {
	// Embeddings contains the vector representations, one per input.
	Embeddings [][]float32
	// Usage tracks resource consumption for this request.
	Usage EmbeddingUsage
	// Model identifies which embedding model was used.
	Model string
}

// ContextualizedEmbeddingResponse contains contextualized embeddings where each chunk
// is embedded with awareness of its surrounding document context.
type ContextualizedEmbeddingResponse struct {
	// DocumentEmbeddings contains embeddings organized by document, then by chunk.
	// Each document is represented as [][]float32 where each inner slice is a chunk embedding.
	DocumentEmbeddings [][][]float32
	// Usage tracks resource consumption for this request.
	Usage EmbeddingUsage
	// Model identifies which embedding model was used.
	Model string
}

// Embedding defines the interface for generating vector embeddings from text and multimodal content.
// It provides methods for standard text embeddings, multimodal embeddings, and contextualized embeddings.
type Embedding interface {
	// GenerateEmbeddings creates vector embeddings from a list of text strings.
	// The optional inputType parameter can specify the intended use ("query", "document", etc.).
	GenerateEmbeddings(
		ctx context.Context,
		texts []string,
		inputType ...string,
	) (*EmbeddingResponse, error)

	// GenerateMultimodalEmbeddings creates embeddings from mixed text and image content.
	// Each input can contain multiple content pieces of different types.
	GenerateMultimodalEmbeddings(
		ctx context.Context,
		inputs []MultimodalInput,
		inputType ...string,
	) (*EmbeddingResponse, error)

	// GenerateContextualizedEmbeddings creates embeddings where each chunk is aware of its document context.
	// Input is organized as documents (outer slice) containing chunks (inner slices).
	GenerateContextualizedEmbeddings(
		ctx context.Context,
		documentChunks [][]string,
		inputType ...string,
	) (*ContextualizedEmbeddingResponse, error)

	// Model returns the embedding model configuration being used.
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
	embed(
		ctx context.Context,
		texts []string,
		inputType ...string,
	) (*EmbeddingResponse, error)
	embedMultimodal(
		ctx context.Context,
		inputs []MultimodalInput,
		inputType ...string,
	) (*EmbeddingResponse, error)
	embedContextualized(
		ctx context.Context,
		documentChunks [][]string,
		inputType ...string,
	) (*ContextualizedEmbeddingResponse, error)
}

type baseEmbedding[C EmbeddingClient] struct {
	options embeddingClientOptions
	client  C
}

// NewEmbedding creates a new embedding client for the specified provider.
// Supported providers include Voyage AI and OpenAI.
// Use WithModel() to specify the embedding model and WithAPIKey() for authentication.
func NewEmbedding(
	provider model.ModelProvider,
	opts ...EmbeddingClientOption,
) (Embedding, error) {
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

func (e *baseEmbedding[C]) GenerateEmbeddings(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embed(ctx, texts, inputType...)
}

func (e *baseEmbedding[C]) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []MultimodalInput,
	inputType ...string,
) (*EmbeddingResponse, error) {
	if len(inputs) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      e.options.model.APIModel,
		}, nil
	}

	return e.client.embedMultimodal(ctx, inputs, inputType...)
}

func (e *baseEmbedding[C]) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*ContextualizedEmbeddingResponse, error) {
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

// WithAPIKey sets the API key for authentication with the embedding provider.
func WithAPIKey(apiKey string) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.apiKey = apiKey
	}
}

// WithModel specifies which embedding model to use for generating embeddings.
func WithModel(model model.EmbeddingModel) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.model = model
	}
}

// WithBatchSize sets the number of texts to process in each batch request.
// Larger batch sizes improve throughput but may increase latency.
func WithBatchSize(batchSize int) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.batchSize = batchSize
	}
}

// WithTimeout sets the maximum duration to wait for embedding requests to complete.
func WithTimeout(timeout time.Duration) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.timeout = &timeout
	}
}

// WithDimensions specifies the output dimensionality for embedding vectors.
// Only supported by models that allow variable dimensions (e.g., OpenAI, Voyage).
func WithDimensions(dimensions int) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.dimensions = &dimensions
	}
}

// WithVoyageOptions applies Voyage AI-specific configuration options.
func WithVoyageOptions(voyageOptions ...VoyageOption) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.voyageOptions = voyageOptions
	}
}

// WithOpenAIOptions applies OpenAI-specific configuration options.
func WithOpenAIOptions(openaiOptions ...OpenAIOption) EmbeddingClientOption {
	return func(options *embeddingClientOptions) {
		options.openaiOptions = openaiOptions
	}
}
