package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

func main() {
	client, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
	)
	if err != nil {
		panic(err)
	}

	staticTemplateExample(client)
	dynamicProviderExample(client)
	conditionalExample(client)
}

func staticTemplateExample(client llm.LLM) {
	a := agent.New(client,
		agent.WithSystemPrompt("You are {{.role}}. The user's name is {{.user_name}}. Be helpful and concise."),
		agent.WithState(map[string]any{
			"role":      "a friendly coding assistant",
			"user_name": "Alice",
		}),
	)

	resp, err := a.Chat(context.Background(), "Hello!")
	if err != nil {
		panic(err)
	}

	fmt.Println("Response:", resp.Content)
	fmt.Println()
}

func dynamicProviderExample(client llm.LLM) {
	a := agent.New(client,
		agent.WithInstructionProvider(func(ctx context.Context, state map[string]any) (string, error) {
			role, _ := state["role"].(string)
			if role == "" {
				role = "an assistant"
			}
			return fmt.Sprintf("You are %s. Current time context: morning. Be brief.", role), nil
		}),
		agent.WithState(map[string]any{
			"role": "a helpful chef",
		}),
	)

	resp, err := a.Chat(context.Background(), "What should I make for breakfast?")
	if err != nil {
		panic(err)
	}

	fmt.Println("Response:", resp.Content)
	fmt.Println()
}

func conditionalExample(client llm.LLM) {
	a := agent.New(client,
		agent.WithSystemPrompt(`You are {{.role}}.
{{if .extra_context}}Additional context: {{.extra_context}}
{{end}}Be concise.`),
		agent.WithState(map[string]any{
			"role": "a math tutor",
		}),
	)

	resp, err := a.Chat(context.Background(), "What is 2+2?")
	if err != nil {
		panic(err)
	}

	fmt.Println("Response:", resp.Content)
}
