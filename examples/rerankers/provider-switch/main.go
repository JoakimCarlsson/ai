package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rerankers"
	rerankercohere "github.com/joakimcarlsson/ai/rerankers/cohere"
	rerankervoyage "github.com/joakimcarlsson/ai/rerankers/voyage"
)

func main() {
	client, provider := newReranker()

	documents := []string{
		"Provider implementations live in separate Go modules.",
		"Batch processors can run requests concurrently.",
		"Rerankers reorder candidate documents by relevance.",
	}
	resp, err := client.Rerank(
		context.Background(),
		"Which document explains provider modules?",
		documents,
	)
	if err != nil {
		log.Fatal(err)
	}

	for i, result := range resp.Results {
		fmt.Printf("[%s] %d. score=%.4f document=%q\n",
			provider, i+1, result.RelevanceScore, result.Document)
	}
}

func newReranker() (rerankers.Reranker, string) {
	switch provider := providerName(); provider {
	case "cohere":
		return rerankercohere.NewReranker(
			rerankercohere.WithAPIKey(requiredEnv("COHERE_API_KEY")),
			rerankercohere.WithModel(
				model.CohereRerankerModels[model.CohereRerank35],
			),
			rerankercohere.WithTopK(3),
			rerankercohere.WithReturnDocuments(true),
		), provider
	case "voyage":
		return rerankervoyage.NewReranker(
			rerankervoyage.WithAPIKey(requiredEnv("VOYAGE_API_KEY")),
			rerankervoyage.WithModel(
				model.VoyageRerankerModels[model.Rerank25Lite],
			),
			rerankervoyage.WithTopK(3),
			rerankervoyage.WithReturnDocuments(true),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use cohere or voyage)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "cohere"
	}
	return provider
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
