package main

import (
	"context"
	"fmt"
	"log"
	"os"

	embeddingvoyage "github.com/joakimcarlsson/ai/embeddings/voyage"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("VOYAGE_API_KEY")
	if apiKey == "" {
		log.Fatal("VOYAGE_API_KEY is required")
	}

	client := embeddingvoyage.NewEmbedding(
		embeddingvoyage.WithAPIKey(apiKey),
		embeddingvoyage.WithModel(
			model.VoyageEmbeddingModels[model.Voyage35Lite],
		),
		embeddingvoyage.WithInputType("document"),
	)

	resp, err := client.GenerateEmbeddings(context.Background(), []string{
		"Go modules let each package declare its own dependencies.",
		"Embeddings turn text into vectors for semantic search.",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("model: %s\n", resp.Model)
	fmt.Printf("vectors: %d\n", len(resp.Embeddings))
	if len(resp.Embeddings) > 0 {
		fmt.Printf("dimensions: %d\n", len(resp.Embeddings[0]))
	}
	fmt.Printf("tokens: %d\n", resp.Usage.TotalTokens)
}
