// Example reranker_cohere demonstrates document reranking with Cohere.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rerankers"
)

func main() {
	reranker, err := rerankers.NewReranker(
		model.ProviderCohere,
		rerankers.WithAPIKey(os.Getenv("COHERE_API_KEY")),
		rerankers.WithModel(
			model.CohereRerankerModels[model.CohereRerank35],
		),
		rerankers.WithReturnDocuments(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	query := "What is machine learning?"
	documents := []string{
		"Machine learning is a subset of artificial intelligence that focuses on algorithms that can learn from data.",
		"The weather today is sunny with a temperature of 25 degrees Celsius.",
		"Deep learning uses neural networks with multiple layers to model and understand complex patterns.",
		"Cooking pasta requires boiling water and adding salt for flavor.",
		"Supervised learning is a type of machine learning where algorithms learn from labeled training data.",
		"Natural language processing enables computers to understand and generate human language.",
	}

	response, err := reranker.Rerank(
		context.Background(),
		query,
		documents,
	)
	if err != nil {
		log.Fatal(err)
	}

	for i, result := range response.Results {
		fmt.Printf(
			"Rank %d (Score: %.4f): %s\n",
			i+1,
			result.RelevanceScore,
			result.Document,
		)
	}
}
