package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

func main() {
	embedder, err := llm.NewEmbedding(model.ProviderVoyage,
		llm.WithEmbeddingAPIKey(""),
		llm.WithEmbeddingModel(model.VoyageEmbeddingModels[model.VoyageMulti3]),
	)
	if err != nil {
		log.Fatal(err)
	}

	multimodalInputs := []llm.MultimodalInput{
		{
			Content: []llm.MultimodalContent{
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
