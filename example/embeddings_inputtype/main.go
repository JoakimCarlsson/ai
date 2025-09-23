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

	embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
		embeddings.WithAPIKey(""),
		embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
	)
	if err != nil {
		log.Fatal(err)
	}

	query := "What is machine learning?"
	documents := []string{
		"Machine learning is a subset of artificial intelligence.",
		"Deep learning uses neural networks with multiple layers.",
		"Natural language processing helps computers understand text.",
	}

	queryEmbedding, err := embedder.GenerateEmbeddings(ctx, []string{query}, "query")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Query embedding: %d dimensions\n", len(queryEmbedding.Embeddings[0]))

	docEmbeddings, err := embedder.GenerateEmbeddings(ctx, documents, "document")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Document embeddings: %d vectors, %d dimensions each\n",
		len(docEmbeddings.Embeddings), len(docEmbeddings.Embeddings[0]))

	regularEmbeddings, err := embedder.GenerateEmbeddings(ctx, []string{"Regular embedding without input type"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Regular embedding: %d dimensions\n", len(regularEmbeddings.Embeddings[0]))
}
