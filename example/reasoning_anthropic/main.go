// Example reasoning_anthropic demonstrates streaming extended thinking from an Anthropic model.
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
		model.ProviderAnthropic,
		llm.WithAPIKey(""),
		llm.WithModel(model.AnthropicModels[model.Claude4Sonnet]),
		llm.WithMaxTokens(16000),
		llm.WithAnthropicOptions(
			llm.WithAnthropicReasoningEffort(llm.AnthropicReasoningEffortMedium),
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
