package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tokens/truncate"
	"github.com/joakimcarlsson/ai/types"
)

const systemPrompt = `You are a helpful assistant. Keep your responses concise.`

func main() {
	ctx := context.Background()

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(1024),
	)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore := session.MemoryStore()

	chatAgent := agent.New(llmClient,
		agent.WithSystemPrompt(systemPrompt),
		agent.WithSession("demo", sessionStore),
		agent.WithContextStrategy(truncate.Strategy(truncate.PreservePairs(), truncate.MinMessages(2)), 500),
	)

	messages := []string{
		"My name is Alice.",
		"I live in New York.",
		"I work as a software engineer.",
		"My favorite color is blue.",
		"What do you know about me?",
	}

	for _, msg := range messages {
		fmt.Printf("User: %s\n", msg)
		response := streamAndCollect(ctx, chatAgent, msg)
		fmt.Printf("Assistant: %s\n\n", response)
	}
}

func streamAndCollect(ctx context.Context, a *agent.Agent, input string) string {
	var sb strings.Builder
	for event := range a.ChatStream(ctx, input) {
		switch event.Type {
		case types.EventContentDelta:
			sb.WriteString(event.Content)
		case types.EventError:
			log.Printf("Error: %v", event.Error)
		}
	}
	return sb.String()
}
