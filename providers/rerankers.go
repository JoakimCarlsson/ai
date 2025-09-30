package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

type RerankerUsage struct {
	TotalTokens int64
}

type RerankerResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document,omitempty"`
}

type RerankerResponse struct {
	Results []RerankerResult
	Usage   RerankerUsage
	Model   string
}

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

type RerankerClientOption func(*rerankerClientOptions)

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

func NewReranker(
	provider model.ModelProvider,
	opts ...RerankerClientOption,
) (Reranker, error) {
	clientOptions := rerankerClientOptions{
		returnDocs: false,
	}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderVoyage:
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

func WithRerankerAPIKey(apiKey string) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.apiKey = apiKey
	}
}

func WithRerankerModel(model model.RerankerModel) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.model = model
	}
}

func WithRerankerTopK(topK int) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.topK = &topK
	}
}

func WithReturnDocuments(returnDocs bool) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.returnDocs = returnDocs
	}
}

func WithRerankerTruncation(truncation bool) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.truncation = &truncation
	}
}

func WithRerankerTimeout(timeout time.Duration) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.timeout = &timeout
	}
}

func WithVoyageRerankerOptions(
	voyageOptions ...VoyageRerankerOption,
) RerankerClientOption {
	return func(options *rerankerClientOptions) {
		options.voyageOptions = voyageOptions
	}
}
