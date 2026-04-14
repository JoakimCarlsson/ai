// Example reasoning_ollama demonstrates streaming thinking output from an Ollama model via the OpenAI-compatible API.
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

	client, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey("ollama"),
		llm.WithModel(model.Model{
			ID:               "qwen3:14b",
			Name:             "Qwen3 14B",
			APIModel:         "qwen3:14b",
			Provider:         model.ProviderOpenAI,
			ContextWindow:    32768,
			DefaultMaxTokens: 4096,
			CanReason:        true,
		}),
		llm.WithMaxTokens(4096),
		llm.WithOpenAIOptions(
			llm.WithOpenAIBaseURL("http://localhost:11434/v1"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	messages := []message.Message{
		message.NewUserMessage("How many r's are in the word 'strawberry'?"),
	}

	for event := range client.StreamResponse(ctx, messages, nil) {
		switch event.Type {
		case types.EventThinkingDelta:
			fmt.Print(event.Thinking)
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventError:
			log.Fatal(event.Error)
		}
	}
}
