// Example batch_concurrent demonstrates batch processing with bounded concurrency.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

func main() {
	ctx := context.Background()

	client, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(200),
	)
	if err != nil {
		log.Fatal(err)
	}

	proc := batch.New(
		batch.WithLLM(client),
		batch.WithMaxConcurrency(5),
		batch.WithProgressCallback(func(p batch.Progress) {
			fmt.Printf(
				"Progress: %d/%d completed, %d failed\n",
				p.Completed, p.Total, p.Failed,
			)
		}),
	)

	requests := []batch.Request{
		{
			ID:   "capital-france",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"What is the capital of France? One word.",
				),
			},
		},
		{
			ID:   "capital-japan",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"What is the capital of Japan? One word.",
				),
			},
		},
		{
			ID:   "capital-brazil",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"What is the capital of Brazil? One word.",
				),
			},
		},
	}

	resp, err := proc.Process(ctx, requests)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nCompleted: %d, Failed: %d\n\n", resp.Completed, resp.Failed)
	for _, r := range resp.Results {
		if r.Err != nil {
			fmt.Printf("[%s] Error: %v\n", r.ID, r.Err)
			continue
		}
		fmt.Printf("[%s] %s\n", r.ID, r.ChatResponse.Content)
	}
}
