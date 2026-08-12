package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/fim"
	fimdeepseek "github.com/joakimcarlsson/ai/fim/deepseek"
	fimmistral "github.com/joakimcarlsson/ai/fim/mistral"
)

func main() {
	client, provider := newFIM()

	prompt := "func multiply(a, b int) int {\n\t"
	suffix := "\n}\n"
	maxTokens := int64(64)
	resp, err := client.Complete(context.Background(), fim.Request{
		Prompt:    prompt,
		Suffix:    suffix,
		MaxTokens: &maxTokens,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s]\n%s%s%s", provider, prompt, resp.Content, suffix)
}

func newFIM() (fim.FIM, string) {
	switch provider := providerName(); provider {
	case "deepseek":
		return fimdeepseek.NewFIM(
			fimdeepseek.WithAPIKey(requiredEnv("DEEPSEEK_API_KEY")),
			fimdeepseek.WithModel(fimdeepseek.Models[fimdeepseek.V32]),
			fimdeepseek.WithMaxTokens(64),
		), provider
	case "mistral":
		return fimmistral.NewFIM(
			fimmistral.WithAPIKey(requiredEnv("MISTRAL_API_KEY")),
			fimmistral.WithModel(fimmistral.Models[fimmistral.Codestral]),
			fimmistral.WithMaxTokens(64),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use mistral or deepseek)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "mistral"
	}
	return provider
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
