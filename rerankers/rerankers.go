// Package rerankers provides a unified interface for document reranking using various AI providers.
//
// Document reranking improves search relevance by reordering a list of documents based on
// their semantic similarity to a query. This package defines the [Reranker] interface and
// the data types that flow through it. Concrete vendor implementations live in subpackages
// (rerankers/voyage, rerankers/cohere); each subpackage exports its own NewReranker
// constructor that returns a tracing-wrapped client implementing the interface.
//
// Example usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/rerankers"
//		"github.com/joakimcarlsson/ai/rerankers/voyage"
//	)
//
//	reranker := voyage.NewReranker(
//		voyage.WithAPIKey("your-api-key"),
//		voyage.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
//		voyage.WithTopK(5),
//		voyage.WithReturnDocuments(true),
//	)
//
//	response, err := reranker.Rerank(ctx, "What is machine learning?",
//		[]string{"...", "..."})
package rerankers

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tracing"
)

// RerankerUsage tracks the resource consumption for reranking operations.
type RerankerUsage struct {
	// TotalTokens is the total number of tokens processed during reranking.
	TotalTokens int64
}

// RerankerResult represents a single document's reranking result with its relevance score.
type RerankerResult struct {
	// Index is the original position of this document in the input list.
	Index int `json:"index"`
	// RelevanceScore indicates how relevant this document is to the query (higher = more relevant).
	RelevanceScore float64 `json:"relevance_score"`
	// Document contains the original document text if WithReturnDocuments(true) was specified.
	Document string `json:"document,omitempty"`
}

// RerankerResponse contains the reranked results and metadata from a reranking request.
type RerankerResponse struct {
	// Results contains the documents ordered by relevance (most relevant first).
	Results []RerankerResult
	// Usage tracks resource consumption for this request.
	Usage RerankerUsage
	// Model identifies which reranker model was used.
	Model string
}

// Reranker defines the interface for document reranking operations.
type Reranker interface {
	// Rerank reorders documents by relevance to the query, returning results sorted by relevance score.
	Rerank(
		ctx context.Context,
		query string,
		documents []string,
	) (*RerankerResponse, error)
	// Model returns the reranker model configuration being used.
	Model() model.RerankerModel
}

// TracingAttrs are construction-time attributes vendor packages forward to the
// [WithTracing] wrapper so they appear on every span produced for the wrapped
// client.
type TracingAttrs struct {
	TopK            *int
	MaxChunksPerDoc *int
	ReturnDocuments *bool
}

// WithTracing wraps a Reranker so every call records OpenTelemetry spans and metrics.
// The attrs are recorded as construction-time span attributes.
func WithTracing(inner Reranker, attrs TracingAttrs) Reranker {
	return &tracingReranker{inner: inner, attrs: attrs}
}

type tracingReranker struct {
	inner Reranker
	attrs TracingAttrs
}

func (t *tracingReranker) Model() model.RerankerModel {
	return t.inner.Model()
}

func (t *tracingReranker) spanAttrs() []tracing.Attr {
	var attrs []tracing.Attr
	if t.attrs.TopK != nil {
		attrs = append(attrs, tracing.AttrRequestTopK.Int(*t.attrs.TopK))
	}
	if t.attrs.MaxChunksPerDoc != nil {
		attrs = append(attrs, tracing.AttrRequestMaxChunksPerDoc.Int(*t.attrs.MaxChunksPerDoc))
	}
	if t.attrs.ReturnDocuments != nil {
		attrs = append(attrs, tracing.AttrRequestReturnDocuments.Bool(*t.attrs.ReturnDocuments))
	}
	return attrs
}

func (t *tracingReranker) Rerank(
	ctx context.Context,
	query string,
	documents []string,
) (*RerankerResponse, error) {
	m := t.inner.Model()
	if len(documents) == 0 {
		return &RerankerResponse{
			Results: []RerankerResult{},
			Usage:   RerankerUsage{TotalTokens: 0},
			Model:   m.APIModel,
		}, nil
	}

	start := time.Now()
	ctx, span := tracing.StartRerankSpan(
		ctx,
		m.APIModel,
		string(m.Provider),
		t.spanAttrs()...,
	)
	defer span.End()
	span.SetAttributes(tracing.AttrDocumentCount.Int(len(documents)))

	resp, err := t.inner.Rerank(ctx, query, documents)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx,
			"rerank",
			m.APIModel,
			string(m.Provider),
			time.Since(start),
			0,
			0,
			err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageTotalTokens.Int64(int64(resp.Usage.TotalTokens)),
		tracing.AttrResultCount.Int(len(resp.Results)),
	)
	tracing.RecordMetrics(
		ctx,
		"rerank",
		m.APIModel,
		string(m.Provider),
		time.Since(start),
		int64(resp.Usage.TotalTokens),
		0,
		nil,
	)
	return resp, nil
}
