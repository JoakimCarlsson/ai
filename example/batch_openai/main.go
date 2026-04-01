// Example batch_openai demonstrates the OpenAI native Batch API with 50% cost savings.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/message"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(option.WithAPIKey(apiKey))

	proc := batch.New(
		batch.WithOpenAIClient(client),
		batch.WithPollInterval(10*time.Second),
		batch.WithProgressCallback(func(p batch.Progress) {
			fmt.Printf(
				"[%s] %d/%d completed, %d failed\n",
				p.Status, p.Completed, p.Total, p.Failed,
			)
		}),
	)

	requests := []batch.Request{
		{
			ID:   "translate-es",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewSystemMessage("You are a translator."),
				message.NewUserMessage("Translate 'hello world' to Spanish."),
			},
		},
		{
			ID:   "translate-fr",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewSystemMessage("You are a translator."),
				message.NewUserMessage("Translate 'hello world' to French."),
			},
		},
		{
			ID:   "translate-de",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewSystemMessage("You are a translator."),
				message.NewUserMessage("Translate 'hello world' to German."),
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
