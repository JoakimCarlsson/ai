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

	// LogProbs is populated by OpenAI (and OpenAI-compatible providers) when
	// WithLogprobs was set; nil otherwise. Print the first few tokens with
	// their log probabilities to show the surface.
	if len(resp.LogProbs) > 0 {
		fmt.Println("\nper-token log probabilities (first 5):")
		for _, tok := range resp.LogProbs[:min(5, len(resp.LogProbs))] {
			fmt.Printf("  %-12q logprob=%.4f\n", tok.Token, tok.LogProb)
		}
	}

	// Choices holds every completion when WithN(>1) was used; it stays empty
	// for a single completion, where the top-level fields above suffice.
	if len(resp.Choices) > 1 {
		fmt.Printf("\n%d choices returned:\n", len(resp.Choices))
		for i, choice := range resp.Choices {
			fmt.Printf("  [%d] %s\n", i, choice.Content)
		}
	}
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
			// Per-token log probabilities (top 3 alternatives per position)
			// surface on resp.LogProbs below. WithN(k) and
			// WithLogitBias(map[string]int{...}) are the other OpenAI-only
			// sampling knobs; WithN populates resp.Choices with every
			// completion.
			llmopenai.WithLogprobs(3),
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
