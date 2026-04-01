// Example batch_openai demonstrates the OpenAI native Batch API.
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
		model.ProviderOpenAI,
		batch.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		batch.WithModel(model.OpenAIModels[model.GPT5Nano]),
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
			ID:   "translate-es",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewSystemMessage("You are a translator."),
				message.NewUserMessage(
					"Translate 'hello world' to Spanish.",
				),
			},
		},
		{
			ID:   "translate-fr",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewSystemMessage("You are a translator."),
				message.NewUserMessage(
					"Translate 'hello world' to French.",
				),
			},
		},
		{
			ID:   "translate-de",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewSystemMessage("You are a translator."),
				message.NewUserMessage(
					"Translate 'hello world' to German.",
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
		fmt.Printf("[%s] %s\n", r.ID, r.ChatResponse.Content)
	}
}
