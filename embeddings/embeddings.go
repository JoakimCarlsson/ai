// Package embeddings provides a unified interface for generating text and multimodal embeddings
// from various AI providers.
//
// This package defines the [Embedding] interface and the data types that flow through it.
// Concrete vendor implementations live in subpackages (embeddings/openai, embeddings/voyage,
// embeddings/cohere, embeddings/gemini, embeddings/bedrock, embeddings/mistral); each subpackage
// exports its own NewEmbedding constructor that returns a tracing-wrapped client implementing
// the interface.
//
// Example usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/embeddings"
//		"github.com/joakimcarlsson/ai/embeddings/voyage"
//	)
//
//	embedder := voyage.NewEmbedding(
//		voyage.WithAPIKey("your-api-key"),
//		voyage.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
//	)
//
//	response, err := embedder.GenerateEmbeddings(ctx, []string{"Hello world"})
package embeddings

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tracing"
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
type Embedding interface {
	// GenerateEmbeddings creates vector embeddings from a list of text strings.
	// The optional inputType parameter can specify the intended use ("query", "document", etc.).
	GenerateEmbeddings(
		ctx context.Context,
		texts []string,
		inputType ...string,
	) (*EmbeddingResponse, error)

	// GenerateMultimodalEmbeddings creates embeddings from mixed text and image content.
	GenerateMultimodalEmbeddings(
		ctx context.Context,
		inputs []MultimodalInput,
		inputType ...string,
	) (*EmbeddingResponse, error)

	// GenerateContextualizedEmbeddings creates embeddings where each chunk is aware of its document context.
	GenerateContextualizedEmbeddings(
		ctx context.Context,
		documentChunks [][]string,
		inputType ...string,
	) (*ContextualizedEmbeddingResponse, error)

	// Model returns the embedding model configuration being used.
	Model() model.EmbeddingModel
}

// TracingAttrs are construction-time attributes vendor packages forward to the
// [WithTracing] wrapper so they appear on every span produced for the wrapped
// client.
type TracingAttrs struct {
	Dimensions *int
}

// WithTracing wraps an Embedding client so every call records OpenTelemetry spans and
// metrics. The attrs are recorded as construction-time span attributes.
func WithTracing(inner Embedding, attrs TracingAttrs) Embedding {
	return &tracingEmbedding{inner: inner, attrs: attrs}
}

type tracingEmbedding struct {
	inner Embedding
	attrs TracingAttrs
}

func (t *tracingEmbedding) Model() model.EmbeddingModel {
	return t.inner.Model()
}

func (t *tracingEmbedding) spanAttrs() []tracing.Attr {
	var attrs []tracing.Attr
	if t.attrs.Dimensions != nil {
		attrs = append(attrs, tracing.AttrRequestDimensions.Int(*t.attrs.Dimensions))
	}
	return attrs
}

func (t *tracingEmbedding) GenerateEmbeddings(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	m := t.inner.Model()
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      m.APIModel,
		}, nil
	}

	start := time.Now()
	ctx, span := tracing.StartEmbeddingSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)
	defer span.End()
	span.SetAttributes(tracing.AttrInputCount.Int(len(texts)))

	resp, err := t.inner.GenerateEmbeddings(ctx, texts, inputType...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "generate_embeddings", m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageTotalTokens.Int64(int64(resp.Usage.TotalTokens)),
	)
	tracing.RecordMetrics(
		ctx, "generate_embeddings", m.APIModel, string(m.Provider),
		time.Since(start), int64(resp.Usage.TotalTokens), 0, nil,
	)
	return resp, nil
}

func (t *tracingEmbedding) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []MultimodalInput,
	inputType ...string,
) (*EmbeddingResponse, error) {
	m := t.inner.Model()
	if len(inputs) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      m.APIModel,
		}, nil
	}

	start := time.Now()
	ctx, span := tracing.StartEmbeddingSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)
	defer span.End()
	span.SetAttributes(tracing.AttrInputCount.Int(len(inputs)))

	resp, err := t.inner.GenerateMultimodalEmbeddings(ctx, inputs, inputType...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "generate_embeddings", m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageTotalTokens.Int64(int64(resp.Usage.TotalTokens)),
	)
	tracing.RecordMetrics(
		ctx, "generate_embeddings", m.APIModel, string(m.Provider),
		time.Since(start), int64(resp.Usage.TotalTokens), 0, nil,
	)
	return resp, nil
}

func (t *tracingEmbedding) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*ContextualizedEmbeddingResponse, error) {
	m := t.inner.Model()
	if len(documentChunks) == 0 {
		return &ContextualizedEmbeddingResponse{
			DocumentEmbeddings: [][][]float32{},
			Usage:              EmbeddingUsage{TotalTokens: 0},
			Model:              m.APIModel,
		}, nil
	}

	start := time.Now()
	ctx, span := tracing.StartEmbeddingSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)
	defer span.End()
	span.SetAttributes(tracing.AttrDocumentCount.Int(len(documentChunks)))

	resp, err := t.inner.GenerateContextualizedEmbeddings(ctx, documentChunks, inputType...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "generate_embeddings", m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageTotalTokens.Int64(int64(resp.Usage.TotalTokens)),
	)
	tracing.RecordMetrics(
		ctx, "generate_embeddings", m.APIModel, string(m.Provider),
		time.Since(start), int64(resp.Usage.TotalTokens), 0, nil,
	)
	return resp, nil
}
