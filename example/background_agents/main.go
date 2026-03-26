// Example background_agents demonstrates orchestrating sub-agents that run research tasks in the background.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
)

func main() {
	ctx := context.Background()

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	researcher := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a concise research assistant. When given a topic, provide a brief 2-3 sentence summary of the key facts.",
		),
	)

	orchestrator := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			`You are a research coordinator. When asked to compare topics:
1. Launch background research tasks for each topic using background: true
2. Then collect all results using get_task_result with wait: true
3. Finally, synthesize the results into a comparison`,
		),
		agent.WithSubAgents(
			agent.SubAgentConfig{
				Name:        "researcher",
				Description: "Research a topic. Supports background: true for async execution.",
				Agent:       researcher,
			},
		),
	)

	fmt.Println("Streaming background agent orchestration:")
	fmt.Println()
	for event := range orchestrator.ChatStream(ctx, "Compare Go and Rust. Research each one in the background, then give me a comparison.") {
		switch event.Type {
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventError:
			log.Fatal(event.Error)
		}
		if event.ToolResult != nil {
			fmt.Printf(
				"\n[Tool: %s → %s]\n",
				event.ToolResult.ToolName,
				truncate(event.ToolResult.Output, 120),
			)
		}
	}
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
