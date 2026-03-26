package embeddings

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type openaiOptions struct {
	baseURL    string
	dimensions *int
	user       string
}

// OpenAIOption configures OpenAI-specific embedding client options.
type OpenAIOption func(*openaiOptions)

type openaiClient struct {
	providerOptions embeddingClientOptions
	options         openaiOptions
	client          openai.Client
}

// OpenAIClient is the OpenAI implementation of EmbeddingClient.
type OpenAIClient EmbeddingClient

func newOpenAIClient(opts embeddingClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{}
	for _, o := range opts.openaiOptions {
		o(&openaiOpts)
	}

	openaiClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		openaiClientOptions = append(
			openaiClientOptions,
			option.WithAPIKey(opts.apiKey),
		)
	}
	if openaiOpts.baseURL != "" {
		openaiClientOptions = append(
			openaiClientOptions,
			option.WithBaseURL(openaiOpts.baseURL),
		)
	}

	client := openai.NewClient(openaiClientOptions...)
	return &openaiClient{
		providerOptions: opts,
		options:         openaiOpts,
		client:          client,
	}
}

func (o *openaiClient) embed(
	ctx context.Context,
	texts []string,
	_ ...string,
) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      o.providerOptions.model.APIModel,
		}, nil
	}

	batchSize := o.providerOptions.batchSize
	if batchSize <= 0 {
		batchSize = 2048
	}

	var allEmbeddings [][]float32
	var totalTokens int64

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		response, err := o.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch: %w", err)
		}

		allEmbeddings = append(allEmbeddings, response.Embeddings...)
		totalTokens += response.Usage.TotalTokens
	}

	return &EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      EmbeddingUsage{TotalTokens: totalTokens},
		Model:      o.providerOptions.model.APIModel,
	}, nil
}

func (o *openaiClient) embedBatch(
	ctx context.Context,
	texts []string,
) (*EmbeddingResponse, error) {
	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(o.providerOptions.model.APIModel),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	}

	if o.providerOptions.dimensions != nil {
		params.Dimensions = openai.Int(int64(*o.providerOptions.dimensions))
	} else if o.options.dimensions != nil {
		params.Dimensions = openai.Int(int64(*o.options.dimensions))
	}
	if o.options.user != "" {
		params.User = openai.String(o.options.user)
	}

	resp, err := o.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embedding := make([]float32, len(data.Embedding))
		for j, val := range data.Embedding {
			embedding[j] = float32(val)
		}
		embeddings[i] = embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Usage: EmbeddingUsage{
			TotalTokens: int64(resp.Usage.TotalTokens),
		},
		Model: string(resp.Model),
	}, nil
}

func (o *openaiClient) embedMultimodal(
	_ context.Context,
	_ []MultimodalInput,
	_ ...string,
) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("OpenAI does not support multimodal embeddings")
}

func (o *openaiClient) embedContextualized(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf("OpenAI does not support contextualized embeddings")
}

// WithBaseURL sets a custom base URL for the OpenAI API endpoint.
// This enables compatibility with Azure OpenAI and other OpenAI-compatible services.
func WithBaseURL(baseURL string) OpenAIOption {
	return func(options *openaiOptions) {
		options.baseURL = baseURL
	}
}

// WithUser sets a unique identifier for the end-user making the request.
// This helps OpenAI monitor and detect abuse.
func WithUser(user string) OpenAIOption {
	return func(options *openaiOptions) {
		options.user = user
	}
}
