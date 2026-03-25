package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// RerankerUsage records token usage for a rerank request.
type RerankerUsage struct {
	TotalTokens int64
}

// RerankerResult is one ranked document in a rerank response.
type RerankerResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document,omitempty"`
}

// RerankerResponse is the full result of a rerank API call.
type RerankerResponse struct {
	Results []RerankerResult
	Usage   RerankerUsage
	Model   string
}

// Reranker scores and orders documents by relevance to a query.
type Reranker interface {
	Rerank(
		ctx context.Context,
		query string,
		documents []string,
	) (*RerankerResponse, error)
	Model() model.RerankerModel
}

type rerankerClientOptions struct {
	apiKey     string
	model      model.RerankerModel
	topK       *int
	returnDocs bool
	truncation *bool
	timeout    *time.Duration

	voyageOptions []VoyageRerankerOption
}

// RerankerClientOption configures construction of a reranker client.
type RerankerClientOption func(*rerankerClientOptions)

// RerankerClient defines the provider-specific implementation for reranking.
type RerankerClient interface {
	rerank(
		ctx context.Context,
		query string,
		documents []string,
	) (*RerankerResponse, error)
}

type baseReranker[C RerankerClient] struct {
	options rerankerClientOptions
	client  C
}

// NewReranker creates a new reranker client for the specified provider.
func NewReranker(
	provider model.Provider,
	opts ...RerankerClientOption,
) (Reranker, error) {
	clientOptions := rerankerClientOptions{
		returnDocs: false,
	}
	for _, o := range opts {
		o(&clientOptions)
	}

	if provider == model.ProviderVoyage {
		return &baseReranker[VoyageRerankerClient]{
			options: clientOptions,
			client:  newVoyageRerankerClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf("reranker provider not supported: %s", provider)
}

func (r *baseReranker[C]) Rerank(
	ctx context.Context,
	query string,
	documents []string,
) (*RerankerResponse, error) {
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

// WithRerankerAPIKey sets the API key for the reranker provider.
func WithRerankerAPIKey(apiKey string) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.apiKey = apiKey
	}
}

// WithRerankerModel specifies which reranker model to use.
func WithRerankerModel(model model.RerankerModel) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.model = model
	}
}

// WithRerankerTopK limits the number of top results returned.
func WithRerankerTopK(topK int) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.topK = &topK
	}
}

// WithReturnDocuments controls whether ranked documents are included in the response.
func WithReturnDocuments(returnDocs bool) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.returnDocs = returnDocs
	}
}

// WithRerankerTruncation enables or disables input truncation for the reranker.
func WithRerankerTruncation(truncation bool) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.truncation = &truncation
	}
}

// WithRerankerTimeout sets the maximum duration for reranker API requests.
func WithRerankerTimeout(timeout time.Duration) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.timeout = &timeout
	}
}

// WithVoyageRerankerOptions applies Voyage-specific reranker configuration options.
func WithVoyageRerankerOptions(
	voyageOptions ...VoyageRerankerOption,
) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.voyageOptions = voyageOptions
	}
}
