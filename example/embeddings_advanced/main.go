package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	ctx := context.Background()
	basicExample(ctx)
	advancedParametersExample(ctx)
	differentDataTypesExample(ctx)
	typeSafetyExample(ctx)
}

func basicExample(ctx context.Context) {
	embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
		embeddings.WithAPIKey(""),
		embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
	)
	if err != nil {
		log.Fatal(err)
	}

	texts := []string{
		"Hello, world!",
		"This is a test document.",
	}

	response, err := embedder.GenerateEmbeddings(ctx, texts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Basic embeddings: %d vectors, %d dimensions each\n",
		len(response.Embeddings), len(response.Embeddings[0]))
}

func advancedParametersExample(ctx context.Context) {
	embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
		embeddings.WithAPIKey(""),
		embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
		embeddings.WithVoyageOptions(
			embeddings.WithInputType("query"),
			embeddings.WithTruncation(false),
			embeddings.WithOutputDimensions(512),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	query := "What is machine learning?"

	response, err := embedder.GenerateEmbeddings(ctx, []string{query})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Query embedding: %d dimensions\n", len(response.Embeddings[0]))
	fmt.Printf("First 5 values: %v\n", response.Embeddings[0][:5])
}

func differentDataTypesExample(ctx context.Context) {
	embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
		embeddings.WithAPIKey(""),
		embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage3Large]),
		embeddings.WithVoyageOptions(
			embeddings.WithInputType("document"),
			embeddings.WithOutputDimensions(256),
			embeddings.WithOutputDtype("int8"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	documents := []string{
		"Document for retrieval with int8 compression.",
		"Another document with reduced precision.",
	}

	response, err := embedder.GenerateEmbeddings(ctx, documents)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Compressed embeddings: %d vectors, %d dimensions each\n",
		len(response.Embeddings), len(response.Embeddings[0]))
	fmt.Printf("First document embedding (first 5): %v\n", response.Embeddings[0][:5])
}

func typeSafetyExample(ctx context.Context) {
	embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
		embeddings.WithAPIKey(""),
		embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
		embeddings.WithVoyageOptions(
			embeddings.WithOutputDtype("uint8"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	text := "Type-safe embedding example with uint8 compression."

	response, err := embedder.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Type-safe embeddings: %d dimensions\n", len(response.Embeddings[0]))
	fmt.Printf("Data automatically converted from uint8 to float32\n")
	fmt.Printf("Sample values: %v\n", response.Embeddings[0][:3])
}
