// Example: provider-native built-in tools.
//
// Set AI_PROVIDER to one of: anthropic, gemini, openai-responses, groq-compound.
// Then provide the matching API key env var. The example asks a question that
// requires the built-in tool to answer correctly (e.g. "what's the latest stable
// Go release?") and prints the assistant content plus any provider metadata.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/llm"
	llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
	llmgemini "github.com/joakimcarlsson/ai/llm/gemini"
	llmgroq "github.com/joakimcarlsson/ai/llm/groq"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

const prompt = "What is the latest stable Go release? Answer in one sentence."

func main() {
	ctx := context.Background()
	client, provider := newLLM()

	messages := []message.Message{
		message.NewUserMessage(prompt),
	}
	resp, err := client.SendMessages(ctx, messages, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] %s\n\n", provider, resp.Content)
	if len(resp.ProviderMetadata) > 0 {
		meta, _ := json.MarshalIndent(resp.ProviderMetadata, "", "  ")
		fmt.Printf("provider metadata:\n%s\n", meta)
	}
}

func newLLM() (llm.LLM, string) {
	switch provider := providerName(); provider {
	case "anthropic":
		return llmanthropic.NewLLM(
			llmanthropic.WithAPIKey(requiredEnv("ANTHROPIC_API_KEY")),
			llmanthropic.WithModel(model.AnthropicModels[model.Claude47Opus]),
			llmanthropic.WithMaxTokens(1024),
			llmanthropic.WithWebSearch(llmanthropic.WebSearchConfig{
				MaxUses: 3,
			}),
		), provider

	case "gemini":
		return llmgemini.NewLLM(
			llmgemini.WithAPIKey(requiredEnv("GEMINI_API_KEY")),
			llmgemini.WithModel(model.GeminiModels[model.Gemini25Flash]),
			llmgemini.WithMaxTokens(1024),
			llmgemini.WithGoogleSearch(),
		), provider

	case "openai-responses":
		return llmopenai.NewResponsesLLM(
			llmopenai.WithResponsesAPIKey(requiredEnv("OPENAI_API_KEY")),
			llmopenai.WithResponsesModel(model.OpenAIModels[model.GPT5]),
			llmopenai.WithResponsesMaxTokens(1024),
			llmopenai.WithWebSearch(llmopenai.WebSearchOpts{
				SearchContextSize: llmopenai.SearchContextMedium,
			}),
		), provider

	case "groq-compound":
		return llmgroq.NewCompoundLLM(
			llmgroq.WithCompoundAPIKey(requiredEnv("GROQ_API_KEY")),
			llmgroq.WithCompoundModel(model.Model{APIModel: "groq/compound"}),
			llmgroq.WithCompoundMaxTokens(1024),
			llmgroq.WithBrowserSearch(),
		), provider

	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use anthropic, gemini, openai-responses, or groq-compound)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "anthropic"
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
