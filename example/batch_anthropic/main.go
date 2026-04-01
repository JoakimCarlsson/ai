// Example batch_anthropic demonstrates the Anthropic Message Batches API.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	ctx := context.Background()

	proc, err := batch.New(
		model.ProviderAnthropic,
		batch.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
		batch.WithModel(model.AnthropicModels[model.Claude4Sonnet]),
		batch.WithMaxTokens(1024),
		batch.WithPollInterval(10*time.Second),
		batch.WithProgressCallback(func(p batch.Progress) {
			fmt.Printf(
				"[%s] %d/%d completed, %d failed\n",
				p.Status, p.Completed, p.Total, p.Failed,
			)
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	requests := []batch.Request{
		{
			ID:   "haiku-nature",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"Write a haiku about nature.",
				),
			},
		},
		{
			ID:   "haiku-tech",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"Write a haiku about technology.",
				),
			},
		},
		{
			ID:   "haiku-ocean",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"Write a haiku about the ocean.",
				),
			},
		},
	}

	resp, err := proc.Process(ctx, requests)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"\nCompleted: %d, Failed: %d\n\n",
		resp.Completed, resp.Failed,
	)
	for _, r := range resp.Results {
		if r.Err != nil {
			fmt.Printf("[%s] Error: %v\n", r.ID, r.Err)
			continue
		}
		fmt.Printf("[%s] %s\n\n", r.ID, r.ChatResponse.Content)
	}
}
