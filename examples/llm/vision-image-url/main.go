package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/llm"
	llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
	llmgemini "github.com/joakimcarlsson/ai/llm/gemini"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

const defaultImageURL = "https://yavuzceliker.github.io/sample-images/image-1.jpg"

func main() {
	client, provider := newLLM()

	userMsg := message.NewUserMessage(
		"Describe this image in two sentences, then name one thing a user might ask next.",
	)
	userMsg.AddImageURL(imageURL(), "low")

	resp, err := client.SendMessages(
		context.Background(),
		[]message.Message{userMsg},
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] %s\n", provider, resp.Content)
}

func imageURL() string {
	url := os.Getenv("IMAGE_URL")
	if url == "" {
		return defaultImageURL
	}
	return url
}

func newLLM() (llm.LLM, string) {
	switch provider := providerName(); provider {
	case "anthropic":
		return llmanthropic.NewLLM(
			llmanthropic.WithAPIKey(requiredEnv("ANTHROPIC_API_KEY")),
			llmanthropic.WithModel(model.AnthropicModels[model.Claude45Haiku]),
			llmanthropic.WithMaxTokens(512),
		), provider
	case "gemini":
		return llmgemini.NewLLM(
			llmgemini.WithAPIKey(requiredEnv("GEMINI_API_KEY")),
			llmgemini.WithModel(model.GeminiModels[model.Gemini25FlashLite]),
			llmgemini.WithMaxTokens(512),
		), provider
	case "openai":
		return llmopenai.NewLLM(
			llmopenai.WithAPIKey(requiredEnv("OPENAI_API_KEY")),
			llmopenai.WithModel(model.OpenAIModels[model.GPT54Nano]),
			llmopenai.WithMaxTokens(512),
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
