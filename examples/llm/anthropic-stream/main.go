package main

import (
	"context"
	"fmt"
	"log"
	"os"

	llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY is required")
	}

	client := llmanthropic.NewLLM(
		llmanthropic.WithAPIKey(apiKey),
		llmanthropic.WithModel(model.AnthropicModels[model.Claude45Haiku]),
		llmanthropic.WithMaxTokens(256),
	)

	events := client.StreamResponse(context.Background(), []message.Message{
		message.NewUserMessage("Write a one-paragraph explanation of streaming responses."),
	}, nil)

	for event := range events {
		if event.Error != nil {
			log.Fatal(event.Error)
		}
		if event.Content != "" {
			fmt.Print(event.Content)
		}
	}
	fmt.Println()
}
