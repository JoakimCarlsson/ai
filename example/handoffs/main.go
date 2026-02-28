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

	billing := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a billing specialist. Help users with invoices, payments, and account balances. Be concise and helpful.",
		),
	)

	support := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a technical support specialist. Help users troubleshoot issues and solve technical problems. Be concise and helpful.",
		),
	)

	triage := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a triage agent. Route users to the appropriate specialist:\n- For billing, payments, or invoice questions: transfer to billing\n- For technical issues or troubleshooting: transfer to support\nDo NOT answer questions yourself — always transfer to a specialist.",
		),
		agent.WithHandoffs(
			agent.HandoffConfig{
				Name:        "billing",
				Description: "For billing, payment, and invoice questions",
				Agent:       billing,
			},
			agent.HandoffConfig{
				Name:        "support",
				Description: "For technical issues and troubleshooting",
				Agent:       support,
			},
		),
	)

	response, err := triage.Chat(ctx, "I was charged twice on my last invoice.")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Agent: %s\n", response.AgentName)
	fmt.Printf("Response: %s\n", response.Content)
}
