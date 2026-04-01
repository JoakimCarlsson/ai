// Example embeddings_gemini demonstrates text embedding generation with Google Gemini.
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
		model.ProviderGemini,
		embeddings.WithAPIKey(
			os.Getenv("GEMINI_API_KEY"),
		),
		embeddings.WithModel(
			model.GeminiEmbeddingModels[model.GeminiTextEmbedding004],
		),
		embeddings.WithGeminiOptions(
			embeddings.WithGeminiTaskType(
				"RETRIEVAL_DOCUMENT",
			),
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
}
