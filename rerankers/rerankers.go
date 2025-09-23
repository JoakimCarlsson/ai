// Package rerankers provides a unified interface for document reranking using various AI providers.
//
// Document reranking improves search relevance by reordering a list of documents based on
// their semantic similarity to a query. This package abstracts the differences between
// reranking providers, offering a consistent API for improving search results.
//
// Key features include:
//   - Document reranking based on semantic similarity
//   - Relevance score calculation for each document
//   - Configurable result count limits (topK)
//   - Optional document content return
//   - Token usage tracking and cost calculation
//   - Provider-specific optimizations
//
// Example usage:
//
//	reranker, err := rerankers.NewReranker(model.ProviderVoyage,
//		rerankers.WithAPIKey("your-api-key"),
//		rerankers.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
//		rerankers.WithTopK(5),
//		rerankers.WithReturnDocuments(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	query := "What is machine learning?"
//	documents := []string{
//		"Machine learning is a subset of artificial intelligence.",
//		"The weather today is sunny.",
//		"Deep learning uses neural networks.",
//	}
//
//	response, err := reranker.Rerank(ctx, query, documents)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for i, result := range response.Results {
//		fmt.Printf("Rank %d (Score: %.4f): %s\n", i+1, result.RelevanceScore, result.Document)
//	}
package rerankers

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
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
	Rerank(ctx context.Context, query string, documents []string) (*RerankerResponse, error)
	// Model returns the reranker model configuration being used.
	Model() model.RerankerModel
}

type rerankerClientOptions struct {
	apiKey     string
	model      model.RerankerModel
	topK       *int
	returnDocs bool
	truncation *bool
	timeout    *time.Duration

	voyageOptions []VoyageOption
}

type RerankerClientOption func(*rerankerClientOptions)

type RerankerClient interface {
	rerank(ctx context.Context, query string, documents []string) (*RerankerResponse, error)
}

type baseReranker[C RerankerClient] struct {
	options rerankerClientOptions
	client  C
}

func NewReranker(provider model.ModelProvider, opts ...RerankerClientOption) (Reranker, error) {
	clientOptions := rerankerClientOptions{
		returnDocs: false,
	}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderVoyage:
		return &baseReranker[VoyageClient]{
			options: clientOptions,
			client:  newVoyageClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf("reranker provider not supported: %s", provider)
}

func (r *baseReranker[C]) Rerank(ctx context.Context, query string, documents []string) (*RerankerResponse, error) {
	if len(documents) == 0 {
		return &RerankerResponse{
			Results: []RerankerResult{},
			Usage:   RerankerUsage{TotalTokens: 0},
			Model:   r.options.model.APIModel,
		}, nil
	}

	return r.client.rerank(ctx, query, documents)
}

func (r *baseReranker[C]) Model() model.RerankerModel {
	return r.options.model
}

func WithAPIKey(apiKey string) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.apiKey = apiKey
	}
}

func WithModel(model model.RerankerModel) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.model = model
	}
}

func WithTopK(topK int) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.topK = &topK
	}
}

func WithReturnDocuments(returnDocs bool) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.returnDocs = returnDocs
	}
}

func WithTruncation(truncation bool) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.truncation = &truncation
	}
}

func WithTimeout(timeout time.Duration) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.timeout = &timeout
	}
}

func WithVoyageOptions(voyageOptions ...VoyageOption) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.voyageOptions = voyageOptions
	}
}
