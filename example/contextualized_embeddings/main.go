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
		embeddings.WithModel(model.VoyageEmbeddingModels[model.VoyageContext3]),
	)
	if err != nil {
		log.Fatal(err)
	}

	documentChunks := [][]string{
		{
			"The quick brown fox jumps over the lazy dog.",
			"This sentence contains every letter of the alphabet at least once.",
		},
		{
			"Machine learning is a subset of artificial intelligence.",
			"It focuses on algorithms that can learn from and make predictions on data.",
		},
	}

	response, err := embedder.GenerateContextualizedEmbeddings(context.Background(), documentChunks)
	if err != nil {
		log.Fatal(err)
	}

	for docIndex, docEmbeddings := range response.DocumentEmbeddings {
		for chunkIndex, chunkEmbedding := range docEmbeddings {
			fmt.Printf("Document %d, Chunk %d: %s\n", docIndex+1, chunkIndex+1, documentChunks[docIndex][chunkIndex])
			fmt.Printf("Dimensions: %d\n", len(chunkEmbedding))
			fmt.Printf("First 5 values: %v\n\n", chunkEmbedding[:5])
		}
	}
}
