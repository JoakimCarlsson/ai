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
	optionalPlaceholderExample(client)
}

func staticTemplateExample(client llm.LLM) {
	a := agent.New(client,
		agent.WithSystemPrompt("You are {role}. The user's name is {user_name}. Be helpful and concise."),
		agent.WithState(map[string]string{
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
		agent.WithInstructionProvider(func(ctx context.Context, state map[string]string) (string, error) {
			role := state["role"]
			if role == "" {
				role = "an assistant"
			}
			return fmt.Sprintf("You are %s. Current time context: morning. Be brief.", role), nil
		}),
		agent.WithState(map[string]string{
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

func optionalPlaceholderExample(client llm.LLM) {
	a := agent.New(client,
		agent.WithSystemPrompt("You are {role}. {extra_context?}Be concise."),
		agent.WithState(map[string]string{
			"role": "a math tutor",
		}),
	)

	resp, err := a.Chat(context.Background(), "What is 2+2?")
	if err != nil {
		panic(err)
	}

	fmt.Println("Response:", resp.Content)
}
