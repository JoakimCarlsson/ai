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
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/tool/functiontool"
)

type weatherArgs struct {
	City string `json:"city"           desc:"City to get weather for"`
	Unit string `json:"unit,omitempty" desc:"Temperature unit"        enum:"celsius,fahrenheit" required:"false"`
}

type weatherReport struct {
	City        string `json:"city"`
	Conditions  string `json:"conditions"`
	Temperature int    `json:"temperature"`
	Unit        string `json:"unit"`
}

func main() {
	ctx := context.Background()
	client, provider := newLLM()

	weatherTool := functiontool.New(
		"get_weather",
		"Get the current weather for a city.",
		getWeather,
	)
	registry := tool.NewRegistry()
	registry.Register(weatherTool)

	messages := []message.Message{
		message.NewSystemMessage("Use tools when they help answer the user."),
		message.NewUserMessage(
			"What is the weather in Stockholm? Answer in one sentence.",
		),
	}

	resp, err := client.SendMessages(ctx, messages, registry.List())
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.ToolCalls) == 0 {
		fmt.Printf("[%s] %s\n", provider, resp.Content)
		return
	}

	assistantMsg := message.NewAssistantMessage()
	assistantMsg.AppendContent(resp.Content)
	assistantMsg.AppendToolCalls(resp.ToolCalls)
	messages = append(messages, assistantMsg)

	toolMsg := message.NewMessage(message.Tool, nil)
	for _, call := range resp.ToolCalls {
		toolResp, err := registry.Execute(ctx, tool.Call{
			ID:    call.ID,
			Name:  call.Name,
			Input: call.Input,
		})
		if err != nil {
			log.Fatal(err)
		}
		toolMsg.AddToolResult(message.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    toolResp.Content,
			Metadata:   toolResp.Metadata,
			IsError:    toolResp.IsError,
		})
	}
	messages = append(messages, toolMsg)

	finalResp, err := client.SendMessages(ctx, messages, registry.List())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] %s\n", provider, finalResp.Content)
}

func getWeather(args weatherArgs) (weatherReport, error) {
	unit := args.Unit
	if unit == "" {
		unit = "celsius"
	}

	temp := 18
	if unit == "fahrenheit" {
		temp = 64
	}

	return weatherReport{
		City:        args.City,
		Conditions:  "partly cloudy",
		Temperature: temp,
		Unit:        unit,
	}, nil
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
			llmopenai.WithParallelToolCalls(false),
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
