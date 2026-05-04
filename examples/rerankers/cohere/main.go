package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/model"
	rerankercohere "github.com/joakimcarlsson/ai/rerankers/cohere"
)

func main() {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		log.Fatal("COHERE_API_KEY is required")
	}

	client := rerankercohere.NewReranker(
		rerankercohere.WithAPIKey(apiKey),
		rerankercohere.WithModel(
			model.CohereRerankerModels[model.CohereRerank35],
		),
		rerankercohere.WithTopK(3),
		rerankercohere.WithReturnDocuments(true),
	)

	documents := []string{
		"Go modules define a module path, dependencies, and version requirements.",
		"Embeddings represent text as vectors for semantic search.",
		"Text-to-speech turns written text into generated audio.",
	}

	resp, err := client.Rerank(
		context.Background(),
		"How do I declare dependencies for a Go package?",
		documents,
	)
	if err != nil {
		log.Fatal(err)
	}

	for i, result := range resp.Results {
		fmt.Printf("%d. score=%.4f index=%d document=%q\n",
			i+1, result.RelevanceScore, result.Index, result.Document)
	}
}
