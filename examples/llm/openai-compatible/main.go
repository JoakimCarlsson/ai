package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, provider := newLLM()

	resp, err := client.SendMessages(context.Background(), []message.Message{
		message.NewUserMessage(
			"Explain OpenAI-compatible APIs in one practical sentence.",
		),
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] %s\n", provider, resp.Content)
}

func newLLM() (llm.LLM, string) {
	switch provider := providerName(); provider {
	case "groq":
		return llmopenai.NewLLM(
			llmopenai.WithAPIKey(requiredFirstEnv("GROQ_API_KEY", "API_KEY")),
			llmopenai.WithBaseURL(
				envOrDefault("BASE_URL", "https://api.groq.com/openai/v1"),
			),
			llmopenai.WithModel(
				modelFromEnv(model.GroqModels[model.Llama3_3_70BVersatile]),
			),
			llmopenai.WithMaxTokens(256),
		), provider
	case "ollama":
		return llmopenai.NewLLM(
			llmopenai.WithAPIKey(os.Getenv("API_KEY")),
			llmopenai.WithBaseURL(
				envOrDefault("BASE_URL", "http://localhost:11434/v1"),
			),
			llmopenai.WithModel(
				modelFromEnv(model.OllamaModels[model.OllamaLlama32_3B]),
			),
			llmopenai.WithMaxTokens(256),
		), provider
	case "openrouter":
		return llmopenai.NewLLM(
			llmopenai.WithAPIKey(
				requiredFirstEnv("OPENROUTER_API_KEY", "API_KEY"),
			),
			llmopenai.WithBaseURL(
				envOrDefault("BASE_URL", "https://openrouter.ai/api/v1"),
			),
			llmopenai.WithModel(
				modelFromEnv(
					model.OpenRouterModels[model.OpenRouterClaude46Sonnet],
				),
			),
			llmopenai.WithMaxTokens(256),
		), provider
	case "custom":
		return llmopenai.NewLLM(
			llmopenai.WithAPIKey(os.Getenv("API_KEY")),
			llmopenai.WithBaseURL(requiredEnv("BASE_URL")),
			llmopenai.WithModel(
				customModel(requiredEnv("MODEL"), model.Provider("custom")),
			),
			llmopenai.WithMaxTokens(256),
		), provider
	case "openai":
		opts := []llmopenai.Option{
			llmopenai.WithAPIKey(requiredFirstEnv("OPENAI_API_KEY", "API_KEY")),
			llmopenai.WithModel(
				modelFromEnv(model.OpenAIModels[model.GPT54Nano]),
			),
			llmopenai.WithMaxTokens(256),
		}
		if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
			opts = append(opts, llmopenai.WithBaseURL(baseURL))
		}
		return llmopenai.NewLLM(opts...), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use openai, groq, openrouter, ollama, or custom)",
			provider,
		)
		return nil, ""
	}
}

func modelFromEnv(defaultModel model.Model) model.Model {
	modelID := os.Getenv("MODEL")
	if modelID == "" {
		return defaultModel
	}
	return customModel(modelID, defaultModel.Provider)
}

func customModel(modelID string, provider model.Provider) model.Model {
	return model.NewCustomModel(
		model.WithModelID(model.ID(modelID)),
		model.WithAPIModel(modelID),
		model.WithName(modelID),
		model.WithProvider(provider),
	)
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "openai"
	}
	return provider
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func requiredFirstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	log.Fatalf("one of %s is required", strings.Join(names, ", "))
	return ""
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
