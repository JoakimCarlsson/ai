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
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
)

type taskSummary struct {
	Title            string `json:"title"             desc:"Short title for the task"`
	Priority         string `json:"priority"          desc:"Priority level"                           enum:"low,medium,high"`
	EstimatedMinutes int    `json:"estimated_minutes" desc:"Estimated time to complete the task"`
	RequiresReview   bool   `json:"requires_review"   desc:"Whether a human should review the result"`
}

func main() {
	client, provider := newLLM()
	if !client.SupportsStructuredOutput() {
		log.Fatalf("%s model does not support structured output", provider)
	}

	outputSchema := schema.NewStructuredOutputFromStruct(
		"task_summary",
		"A normalized project task summary.",
		taskSummary{},
	)

	resp, err := client.SendMessagesWithStructuredOutput(
		context.Background(),
		[]message.Message{
			message.NewUserMessage(
				"Summarize this task: Add provider-switching LLM examples for tools and JSON output.",
			),
		},
		nil,
		outputSchema,
	)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StructuredOutput == nil {
		log.Fatal("model did not return structured output")
	}

	var summary taskSummary
	if err := json.Unmarshal(
		[]byte(*resp.StructuredOutput),
		&summary,
	); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] %+v\n", provider, summary)
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
