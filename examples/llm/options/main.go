package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
	llmgemini "github.com/joakimcarlsson/ai/llm/gemini"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, provider := newLLM()

	resp, err := client.SendMessages(context.Background(), []message.Message{
		message.NewUserMessage(
			"Give exactly three terse tips for writing maintainable Go. Stop after the third tip.",
		),
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] %s\n", provider, resp.Content)
}

func newLLM() (llm.LLM, string) {
	timeout := 30 * time.Second

	switch provider := providerName(); provider {
	case "anthropic":
		return llmanthropic.NewLLM(
			llmanthropic.WithAPIKey(requiredEnv("ANTHROPIC_API_KEY")),
			llmanthropic.WithModel(model.AnthropicModels[model.Claude45Haiku]),
			llmanthropic.WithMaxTokens(256),
			llmanthropic.WithTemperature(0.2),
			llmanthropic.WithTopP(0.9),
			llmanthropic.WithStopSequences("\n\n"),
			llmanthropic.WithTimeout(timeout),
		), provider
	case "gemini":
		return llmgemini.NewLLM(
			llmgemini.WithAPIKey(requiredEnv("GEMINI_API_KEY")),
			llmgemini.WithModel(model.GeminiModels[model.Gemini25FlashLite]),
			llmgemini.WithMaxTokens(256),
			llmgemini.WithTemperature(0.2),
			llmgemini.WithTopP(0.9),
			llmgemini.WithStopSequences("\n\n"),
			llmgemini.WithTimeout(timeout),
		), provider
	case "openai":
		return llmopenai.NewLLM(
			llmopenai.WithAPIKey(requiredEnv("OPENAI_API_KEY")),
			llmopenai.WithModel(model.OpenAIModels[model.GPT54Nano]),
			llmopenai.WithMaxTokens(256),
			llmopenai.WithTemperature(0.2),
			llmopenai.WithTopP(0.9),
			llmopenai.WithStopSequences("\n\n"),
			llmopenai.WithSeed(42),
			llmopenai.WithTimeout(timeout),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use openai, anthropic, or gemini)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "openai"
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
