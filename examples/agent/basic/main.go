package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	llmClient := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT4oMini]),
		llmopenai.WithMaxTokens(256),
	)

	assistant := agent.New(llmClient,
		agent.WithSystemPrompt("You explain Go concepts clearly and briefly."),
	)

	resp, err := assistant.Chat(
		context.Background(),
		"Why would a library split providers into separate Go modules?",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Content)
}
