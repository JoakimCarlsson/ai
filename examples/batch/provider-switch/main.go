package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/batch"
	batchconcurrent "github.com/joakimcarlsson/ai/batch/concurrent"
	"github.com/joakimcarlsson/ai/llm"
	llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
	llmgemini "github.com/joakimcarlsson/ai/llm/gemini"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	llmClient, provider := newLLM()
	processor := batchconcurrent.NewProcessor(
		batchconcurrent.WithLLM(llmClient),
		batchconcurrent.WithMaxConcurrency(2),
	)

	resp, err := processor.Process(context.Background(), []batch.Request{
		{
			ID:   "one",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage(
					"Say one benefit of provider interfaces.",
				),
			},
		},
		{
			ID:   "two",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("Say one benefit of small Go modules."),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, result := range resp.Results {
		if result.Err != nil {
			fmt.Printf("[%s] %s: error: %v\n", provider, result.ID, result.Err)
			continue
		}
		fmt.Printf(
			"[%s] %s: %s\n",
			provider,
			result.ID,
			result.ChatResponse.Content,
		)
	}
}

func newLLM() (llm.LLM, string) {
	switch provider := providerName(); provider {
	case "anthropic":
		return llmanthropic.NewLLM(
			llmanthropic.WithAPIKey(requiredEnv("ANTHROPIC_API_KEY")),
			llmanthropic.WithModel(model.AnthropicModels[model.Claude45Haiku]),
			llmanthropic.WithMaxTokens(128),
		), provider
	case "gemini":
		return llmgemini.NewLLM(
			llmgemini.WithAPIKey(requiredEnv("GEMINI_API_KEY")),
			llmgemini.WithModel(model.GeminiModels[model.Gemini25FlashLite]),
			llmgemini.WithMaxTokens(128),
		), provider
	case "openai":
		return llmopenai.NewLLM(
			llmopenai.WithAPIKey(requiredEnv("OPENAI_API_KEY")),
			llmopenai.WithModel(model.OpenAIModels[model.GPT54Nano]),
			llmopenai.WithMaxTokens(128),
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
