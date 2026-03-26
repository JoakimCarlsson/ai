// Example fan_out demonstrates parallel sub-agent research via agent fan-out tooling.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
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
			"You are a research coordinator. When asked to compare topics, use the parallel_research tool to investigate all topics simultaneously, then synthesize the results into a comparison.",
		),
		agent.WithFanOut(agent.FanOutConfig{
			Name:           "parallel_research",
			Description:    "Research multiple topics in parallel. Each task string should be a specific research question.",
			Agent:          researcher,
			MaxConcurrency: 3,
		}),
	)

	response, err := orchestrator.Chat(
		ctx,
		"Compare these programming languages: Go, Rust, and Zig. Research each one.",
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
