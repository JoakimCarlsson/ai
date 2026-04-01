// Example embeddings_cohere demonstrates text embedding generation with Cohere.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	embedder, err := embeddings.NewEmbedding(
		model.ProviderCohere,
		embeddings.WithAPIKey(
			os.Getenv("COHERE_API_KEY"),
		),
		embeddings.WithModel(
			model.CohereEmbeddingModels[model.CohereEmbedEnV3],
		),
		embeddings.WithCohereOptions(
			embeddings.WithCohereInputType("search_document"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	texts := []string{
		"Hello world",
		"How are you?",
		"Machine learning is fascinating",
	}

	response, err := embedder.GenerateEmbeddings(
		context.Background(),
		texts,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Generated %d embeddings\n",
		len(response.Embeddings),
	)
	for i, emb := range response.Embeddings {
		fmt.Printf(
			"Text %d: %d dimensions\n",
			i+1,
			len(emb),
		)
	}
	fmt.Printf(
		"Total tokens: %d\n",
		response.Usage.TotalTokens,
	)
}
