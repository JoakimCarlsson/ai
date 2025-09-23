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
		embeddings.WithModel(model.VoyageEmbeddingModels[model.VoyageMulti3]),
	)
	if err != nil {
		log.Fatal(err)
	}

	multimodalInputs := []embeddings.MultimodalInput{
		{
			Content: []embeddings.MultimodalContent{
				{
					Type: "text",
					Text: "This is a banana.",
				},
				{
					Type:     "image_url",
					ImageURL: "https://raw.githubusercontent.com/voyage-ai/voyage-multimodal-3/refs/heads/main/images/banana.jpg",
				},
			},
		},
	}

	response, err := embedder.GenerateMultimodalEmbeddings(context.Background(), multimodalInputs)
	if err != nil {
		log.Fatal(err)
	}

	if len(response.Embeddings) > 0 {
		fmt.Printf("Multimodal embedding dimensions: %d\n", len(response.Embeddings[0]))
		fmt.Printf("First 5 values: %v\n", response.Embeddings[0][:5])
	}
}
