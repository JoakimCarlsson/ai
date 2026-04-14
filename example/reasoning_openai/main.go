// Example reasoning_openai demonstrates configuring reasoning effort for an OpenAI o-series model.
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
		llm.WithAPIKey(""),
		llm.WithModel(model.OpenAIModels[model.O4Mini]),
		llm.WithMaxTokens(16000),
		llm.WithOpenAIOptions(
			llm.WithReasoningEffort(llm.OpenAIReasoningEffortMedium),
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
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventError:
			log.Fatal(event.Error)
		}
	}
}
