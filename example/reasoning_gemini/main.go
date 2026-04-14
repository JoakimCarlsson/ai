// Example reasoning_gemini demonstrates streaming thinking output from a Gemini model.
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
		model.ProviderGemini,
		llm.WithAPIKey(""),
		llm.WithModel(model.GeminiModels[model.Gemini3Pro]),
		llm.WithMaxTokens(16000),
		llm.WithGeminiOptions(
			llm.WithGeminiThinkingLevel(llm.GeminiThinkingLevelMedium),
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
