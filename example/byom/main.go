package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
)

func main() {
	ctx := context.Background()

	customModel := model.NewCustomModel(
		model.WithModelID("gpt-oss-20b"),
		model.WithAPIModel("openai/gpt-oss-20b"),
		model.WithName("GPT OSS 20B"),
		model.WithContextWindow(131_000),
		model.WithDefaultMaxTokens(4096),
		model.WithStructuredOutput(false),
	)

	provider := llm.RegisterCustomProvider("custom", llm.CustomProviderConfig{
		BaseURL:      "http://127.0.0.1:1234/v1",
		DefaultModel: customModel,
	})

	client, err := llm.NewLLM(provider,
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	messages := []message.Message{
		message.NewUserMessage("Explain what BYOM means in a very detailed way in a ai scenario."),
	}

	stream := client.StreamResponse(ctx, messages, nil)

	for event := range stream {
		switch event.Type {
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventComplete:
			fmt.Println()
		case types.EventError:
			log.Fatal(event.Error)
		}
	}
}
