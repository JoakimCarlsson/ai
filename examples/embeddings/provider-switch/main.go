package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/embeddings"
	embeddingcohere "github.com/joakimcarlsson/ai/embeddings/cohere"
	embeddingopenai "github.com/joakimcarlsson/ai/embeddings/openai"
	embeddingvoyage "github.com/joakimcarlsson/ai/embeddings/voyage"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, provider := newEmbedding()

	resp, err := client.GenerateEmbeddings(context.Background(), []string{
		"Provider interfaces make it easy to swap implementations.",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] model=%s vectors=%d dimensions=%d\n",
		provider, resp.Model, len(resp.Embeddings), len(resp.Embeddings[0]))
}

func newEmbedding() (embeddings.Embedding, string) {
	switch provider := providerName(); provider {
	case "cohere":
		return embeddingcohere.NewEmbedding(
			embeddingcohere.WithAPIKey(requiredEnv("COHERE_API_KEY")),
			embeddingcohere.WithModel(
				model.CohereEmbeddingModels[model.CohereEmbedEnV3],
			),
			embeddingcohere.WithInputType("search_document"),
			embeddingcohere.WithEmbeddingTypes([]string{"float"}),
		), provider
	case "openai":
		return embeddingopenai.NewEmbedding(
			embeddingopenai.WithAPIKey(requiredEnv("OPENAI_API_KEY")),
			embeddingopenai.WithModel(
				model.OpenAIEmbeddingModels[model.TextEmbedding3Small],
			),
		), provider
	case "voyage":
		return embeddingvoyage.NewEmbedding(
			embeddingvoyage.WithAPIKey(requiredEnv("VOYAGE_API_KEY")),
			embeddingvoyage.WithModel(
				model.VoyageEmbeddingModels[model.Voyage35Lite],
			),
			embeddingvoyage.WithInputType("document"),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use openai, voyage, or cohere)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "openai"
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
