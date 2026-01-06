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
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tokens/sliding"
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
		agent.WithContextStrategy(sliding.Strategy(sliding.KeepLast(4)), 300),
	)

	counter, _ := tokens.NewCounter()

	messages := []string{
		"My name is Alice.",
		"I live in New York.",
		"I work as a software engineer.",
		"My favorite color is blue.",
		"I have a cat named Whiskers.",
		"What do you know about me?",
	}

	for i, msg := range messages {
		fmt.Printf("\n--- Turn %d ---\n", i+1)
		fmt.Printf("User: %s\n", msg)

		llmMsgs, _ := chatAgent.PeekContextMessages(ctx, msg)

		response := streamAndCollect(ctx, chatAgent, msg)
		fmt.Printf("Assistant: %s\n", response)

		sess, _ := sessionStore.Load(ctx, "demo")
		sessionMsgs, _ := sess.GetMessages(ctx, nil)

		llmCount, _ := counter.CountTokens(ctx, tokens.CountOptions{Messages: llmMsgs, SystemPrompt: systemPrompt})
		sessionCount, _ := counter.CountTokens(
			ctx,
			tokens.CountOptions{Messages: sessionMsgs, SystemPrompt: systemPrompt},
		)
		fmt.Printf(
			"[LLM received: %d msgs, %d tokens] [Session stored: %d msgs, %d tokens]\n",
			len(llmMsgs),
			llmCount.TotalTokens,
			len(sessionMsgs),
			sessionCount.TotalTokens,
		)
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
