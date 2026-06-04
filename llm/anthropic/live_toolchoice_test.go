//go:build live

package anthropic

import (
	"context"
	"os"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
)

type liveWeatherTool struct{}

func (liveWeatherTool) Info() tool.Info {
	return tool.Info{
		Name:        "get_weather",
		Description: "Get the current weather for a city.",
		Parameters: map[string]any{
			"city": map[string]any{"type": "string", "description": "City name"},
		},
		Required: []string{"city"},
	}
}

func (liveWeatherTool) Run(context.Context, tool.Call) (tool.Response, error) {
	return tool.Response{Content: "sunny, 20C"}, nil
}

func liveAnthropic(t *testing.T, tc llm.ToolChoice) llm.LLM {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	return NewLLM(
		WithAPIKey(key),
		WithModel(model.AnthropicModels[model.Claude45Haiku]),
		WithMaxTokens(256),
		WithToolChoice(tc),
	)
}

func send(t *testing.T, c llm.LLM, prompt string) *llm.Response {
	t.Helper()
	resp, err := c.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage(prompt)},
		[]tool.BaseTool{liveWeatherTool{}})
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	return resp
}

func TestLiveAnthropicRequired(t *testing.T) {
	resp := send(t, liveAnthropic(t, llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
		"Say hello in one word.")
	if len(resp.ToolCalls) == 0 {
		t.Fatalf("Required: expected a tool call, got content=%q", resp.Content)
	}
	t.Logf("Required: tool calls=%d first=%s", len(resp.ToolCalls), resp.ToolCalls[0].Name)
}

func TestLiveAnthropicNone(t *testing.T) {
	resp := send(t, liveAnthropic(t, llm.ToolChoice{Mode: llm.ToolChoiceNone}),
		"What is the weather in Paris? Use the tool.")
	if len(resp.ToolCalls) != 0 {
		t.Fatalf("None: expected no tool call, got %d", len(resp.ToolCalls))
	}
	t.Logf("None: content=%q", resp.Content)
}

func TestLiveAnthropicSpecific(t *testing.T) {
	resp := send(t, liveAnthropic(t,
		llm.ToolChoice{Mode: llm.ToolChoiceSpecific, Name: "get_weather"}),
		"Say hello in one word.")
	if len(resp.ToolCalls) == 0 || resp.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("Specific: expected get_weather call, got %+v", resp.ToolCalls)
	}
	t.Logf("Specific: called %s", resp.ToolCalls[0].Name)
}
