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
	"github.com/joakimcarlsson/ai/types"
)

type bugReport struct {
	Title    string `json:"title"    desc:"Short bug title"`
	Severity string `json:"severity" desc:"Bug severity"                     enum:"low,medium,high,critical"`
	Area     string `json:"area"     desc:"System area most likely affected"`
}

func main() {
	client, provider := newLLM()
	if !client.SupportsStructuredOutput() {
		log.Fatalf("%s model does not support structured output", provider)
	}

	outputSchema := schema.NewStructuredOutputFromStruct(
		"bug_report",
		"A normalized bug report.",
		bugReport{},
	)

	events := client.StreamResponseWithStructuredOutput(
		context.Background(),
		[]message.Message{
			message.NewUserMessage(
				"Turn this into a bug report: streaming JSON sometimes stops before the final brace.",
			),
		},
		nil,
		outputSchema,
	)

	fmt.Printf("[%s] streaming JSON: ", provider)
	var finalResp *llm.Response
	for event := range events {
		switch event.Type {
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventComplete:
			finalResp = event.Response
		case types.EventError:
			log.Fatal(event.Error)
		}
	}
	fmt.Println()

	if finalResp == nil || finalResp.StructuredOutput == nil {
		log.Fatal("model did not return final structured output")
	}

	var report bugReport
	if err := json.Unmarshal([]byte(*finalResp.StructuredOutput), &report); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[%s] parsed: %+v\n", provider, report)
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
