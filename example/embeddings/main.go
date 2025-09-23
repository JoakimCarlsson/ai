package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
		embeddings.WithAPIKey(""),
		embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
	)
	if err != nil {
		log.Fatal(err)
	}

	texts := []string{
		"Hello, world!",
		"This is a test document for embedding generation.",
		"Machine learning and natural language processing are fascinating fields.",
	}

	response, err := embedder.GenerateEmbeddings(context.Background(), texts)
	if err != nil {
		log.Fatal(err)
	}

	for i, embedding := range response.Embeddings {
		fmt.Printf("Text: %s\n", texts[i])
		fmt.Printf("Dimensions: %d\n", len(embedding))
		fmt.Printf("First 5 values: %v\n\n", embedding[:5])
	}
}
