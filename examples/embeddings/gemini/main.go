package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joakimcarlsson/ai/embeddings"
	geminiembed "github.com/joakimcarlsson/ai/embeddings/gemini"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "GEMINI_API_KEY not set")
		os.Exit(1)
	}
	embedder := geminiembed.NewEmbedding(
		geminiembed.WithAPIKey(apiKey),
		geminiembed.WithModel(
			model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		),
		geminiembed.WithDimensions(1536),
	)

	imgBytes, err := os.ReadFile("black-dog.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading black-dog.png: %v\n", err)
		os.Exit(1)
	}

	resp, err := embedder.GenerateMultimodalEmbeddings(
		ctx,
		[]embeddings.MultimodalInput{
			{
				Content: []embeddings.MultimodalContent{
					{ContentData: imgBytes, MimeType: "image/png"},
					{Type: "text", Text: "a cute black dog"},
				},
			},
		},
		"RETRIEVAL_DOCUMENT",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generating multimodal embedding: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"got %d embeddings, dim=%d\n",
		len(resp.Embeddings),
		len(resp.Embeddings[0]),
	)

}
