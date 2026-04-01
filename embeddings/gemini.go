package embeddings

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type geminiOptions struct {
	taskType string
}

// GeminiOption configures Gemini-specific embedding behavior.
type GeminiOption func(*geminiOptions)

type geminiClient struct {
	providerOptions embeddingClientOptions
	options         geminiOptions
	client          *genai.Client
}

// GeminiClient is the Gemini implementation of EmbeddingClient.
type GeminiClient EmbeddingClient

func newGeminiClient(
	opts embeddingClientOptions,
) GeminiClient {
	geminiOpts := geminiOptions{}
	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey:  opts.apiKey,
			Backend: genai.BackendGeminiAPI,
		},
	)
	if err != nil {
		return nil
	}

	return &geminiClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
	}
}

func (g *geminiClient) embed(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      g.providerOptions.model.APIModel,
		}, nil
	}

	batchSize := g.providerOptions.batchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		resp, err := g.embedBatch(
			ctx,
			texts[i:end],
			inputType...,
		)
		if err != nil {
			return nil, err
		}

		allEmbeddings = append(
			allEmbeddings,
			resp.Embeddings...,
		)
	}

	return &EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      EmbeddingUsage{},
		Model:      g.providerOptions.model.APIModel,
	}, nil
}

func (g *geminiClient) embedBatch(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = genai.NewContentFromText(text, "user")
	}

	config := &genai.EmbedContentConfig{}
	taskType := g.options.taskType
	if len(inputType) > 0 && inputType[0] != "" {
		taskType = inputType[0]
	}
	if taskType != "" {
		config.TaskType = taskType
	}
	if g.providerOptions.dimensions != nil {
		dim := int32(*g.providerOptions.dimensions)
		config.OutputDimensionality = &dim
	}

	modelName := g.providerOptions.model.APIModel
	resp, err := g.client.Models.EmbedContent(
		ctx,
		modelName,
		contents,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate embeddings: %w",
			err,
		)
	}

	embeddings := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		embeddings[i] = emb.Values
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Usage:      EmbeddingUsage{},
		Model:      modelName,
	}, nil
}

func (g *geminiClient) embedMultimodal(
	_ context.Context,
	_ []MultimodalInput,
	_ ...string,
) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"gemini does not support multimodal embeddings",
	)
}

func (g *geminiClient) embedContextualized(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"gemini does not support contextualized embeddings",
	)
}

// WithGeminiTaskType sets the task type for embeddings (e.g., "RETRIEVAL_DOCUMENT", "RETRIEVAL_QUERY").
func WithGeminiTaskType(taskType string) GeminiOption {
	return func(options *geminiOptions) {
		options.taskType = taskType
	}
}
