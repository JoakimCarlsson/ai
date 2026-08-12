package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/llm/openai"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	client := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(openai.Models[openai.GPT54Nano]),
		llmopenai.WithMaxTokens(256),
	)

	resp, err := client.SendMessages(context.Background(), []message.Message{
		message.NewUserMessage("Explain Go modules in two short sentences."),
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Content)
}
