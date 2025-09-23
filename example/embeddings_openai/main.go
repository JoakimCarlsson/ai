package main

import (
	"context"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	ctx := context.Background()

	embedder, err := embeddings.NewEmbedding(model.ProviderOpenAI,
		embeddings.WithAPIKey(""),
		embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Large]),
		embeddings.WithOpenAIOptions(
			embeddings.WithDimensions(1024),
		),
	)
	if err != nil {
		panic(err)
	}

	texts := []string{
		"Hello, world!",
		"This is a test document.",
		"OpenAI embeddings are working!",
	}

	response, err := embedder.GenerateEmbeddings(ctx, texts)
	if err != nil {
		panic(err)
	}

	_ = response
}
