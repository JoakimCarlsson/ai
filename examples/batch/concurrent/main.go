package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/batch"
	batchconcurrent "github.com/joakimcarlsson/ai/batch/concurrent"
	llmgemini "github.com/joakimcarlsson/ai/llm/gemini"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY is required")
	}

	llmClient := llmgemini.NewLLM(
		llmgemini.WithAPIKey(apiKey),
		llmgemini.WithModel(model.GeminiModels[model.Gemini25FlashLite]),
		llmgemini.WithMaxTokens(128),
	)

	processor := batchconcurrent.NewProcessor(
		batchconcurrent.WithLLM(llmClient),
		batchconcurrent.WithMaxConcurrency(2),
	)

	resp, err := processor.Process(context.Background(), []batch.Request{
		{
			ID:   "modules",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"What is a Go module? Answer in one sentence.",
				),
			},
		},
		{
			ID:   "packages",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"What is a Go package? Answer in one sentence.",
				),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, result := range resp.Results {
		if result.Err != nil {
			fmt.Printf("%s: error: %v\n", result.ID, result.Err)
			continue
		}
		fmt.Printf("%s: %s\n", result.ID, result.ChatResponse.Content)
	}
}
