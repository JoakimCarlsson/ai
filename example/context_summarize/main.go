package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tokens/summarize"
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

	sessionStore := session.FileStore("./sessions")

	chatAgent := agent.New(llmClient,
		agent.WithSystemPrompt(systemPrompt),
		agent.WithSession("summarize-demo", sessionStore),
		agent.WithContextStrategy(summarize.Strategy(llmClient, summarize.KeepRecent(2)), 300),
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

		response := streamAndCollect(ctx, chatAgent, msg)
		fmt.Printf("Assistant: %s\n", response)

		sess, _ := sessionStore.Load(ctx, "summarize-demo")
		sessionMsgs, _ := sess.GetMessages(ctx, nil)

		allMsgs := append([]message.Message{message.NewSystemMessage(systemPrompt)}, sessionMsgs...)
		count, _ := counter.CountTokens(ctx, tokens.CountOptions{Messages: allMsgs, SystemPrompt: systemPrompt})
		fmt.Printf("[Session: %d messages, %d tokens, limit: 300]\n", len(sessionMsgs), count.TotalTokens)
	}

	fmt.Println("\n=== FINAL SESSION STATE ===")
	sess, _ := sessionStore.Load(ctx, "summarize-demo")
	finalMsgs, _ := sess.GetMessages(ctx, nil)

	for i, msg := range finalMsgs {
		content := msg.Content().Text
		if len(content) > 80 {
			content = content[:80] + "..."
		}
		fmt.Printf("[%d] %s: %s\n", i, msg.Role, content)
	}

	data, _ := json.MarshalIndent(finalMsgs, "", "  ")
	os.WriteFile("./sessions/summarize-demo.json", data, 0644)
	fmt.Println("\nSession saved to ./sessions/summarize-demo.json")
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
