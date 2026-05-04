package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/fim"
	fimmistral "github.com/joakimcarlsson/ai/fim/mistral"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		log.Fatal("MISTRAL_API_KEY is required")
	}

	maxTokens := int64(64)
	client := fimmistral.NewFIM(
		fimmistral.WithAPIKey(apiKey),
		fimmistral.WithModel(model.MistralModels[model.Codestral]),
		fimmistral.WithMaxTokens(maxTokens),
	)

	resp, err := client.Complete(context.Background(), fim.Request{
		Prompt:    "func add(a, b int) int {\n\t",
		Suffix:    "\n}\n",
		MaxTokens: &maxTokens,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s%s%s", "func add(a, b int) int {\n\t", resp.Content, "\n}\n")
}
