package berget

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

func key(t *testing.T) string {
	t.Helper()
	k := os.Getenv("BERGET_API_KEY")
	if k == "" {
		t.Skip("set BERGET_API_KEY")
	}
	return k
}

func client(t *testing.T, id model.ID) llm.LLM {
	return NewLLM(
		llmopenai.WithAPIKey(key(t)),
		llmopenai.WithModel(model.BergetModels[id]),
		llmopenai.WithMaxTokens(64),
	)
}

func TestLive(t *testing.T) {
	c := client(t, model.BergetMistralSmall32)
	resp, err := c.SendMessages(context.Background(), []message.Message{
		message.NewUserMessage("Reply with exactly the word: pong"),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("content=%q in=%d out=%d", resp.Content, resp.Usage.InputTokens, resp.Usage.OutputTokens)
	if resp.Content == "" {
		t.Fatal("empty content")
	}
}

func TestLiveStreaming(t *testing.T) {
	c := client(t, model.BergetMistralSmall32)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var b strings.Builder
	var deltas int
	var done bool
	for ev := range c.StreamResponse(ctx, []message.Message{
		message.NewUserMessage("Count: one two three"),
	}, nil) {
		switch ev.Type {
		case types.EventContentDelta:
			deltas++
			b.WriteString(ev.Content)
		case types.EventComplete:
			done = true
		case types.EventError:
			t.Fatalf("stream error: %v", ev.Error)
		}
	}
	t.Logf("deltas=%d done=%v text=%q", deltas, done, b.String())
	if !done || b.Len() == 0 {
		t.Fatalf("stream incomplete: done=%v len=%d", done, b.Len())
	}
}

type weatherTool struct{}

func (weatherTool) Info() tool.Info {
	return tool.Info{
		Name:        "get_weather",
		Description: "Get the current weather for a city",
		Parameters: map[string]any{
			"city": map[string]any{"type": "string", "description": "City name"},
		},
		Required: []string{"city"},
	}
}

func (weatherTool) Run(_ context.Context, _ tool.Call) (tool.Response, error) {
	return tool.NewTextResponse("sunny, 22C"), nil
}

func TestLiveToolCalling(t *testing.T) {
	c := client(t, model.BergetMistralSmall32)
	resp, err := c.SendMessages(context.Background(), []message.Message{
		message.NewUserMessage("What is the weather in Paris? Use the get_weather tool."),
	}, []tool.BaseTool{weatherTool{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("toolCalls=%d content=%q", len(resp.ToolCalls), resp.Content)
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected a tool call, got none")
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("tool name = %q, want get_weather", resp.ToolCalls[0].Name)
	}
}

func TestLiveStructuredOutput(t *testing.T) {
	c := client(t, model.BergetMistralSmall32)
	out := schema.NewStructuredOutputInfo(
		"capital",
		"The capital city of a country",
		map[string]any{
			"country": map[string]any{"type": "string"},
			"capital": map[string]any{"type": "string"},
		},
		[]string{"country", "capital"},
	)
	resp, err := c.SendMessagesWithStructuredOutput(context.Background(), []message.Message{
		message.NewUserMessage("What is the capital of France?"),
	}, nil, out)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredOutput == nil {
		t.Fatal("no structured output returned")
	}
	t.Logf("native=%v json=%s", resp.UsedNativeStructuredOutput, *resp.StructuredOutput)
	var got struct{ Country, Capital string }
	if err := json.Unmarshal([]byte(*resp.StructuredOutput), &got); err != nil {
		t.Fatalf("structured output not valid JSON: %v", err)
	}
	if !strings.EqualFold(got.Capital, "Paris") {
		t.Fatalf("capital = %q, want Paris", got.Capital)
	}
}

func TestLiveAllChatModels(t *testing.T) {
	key(t)
	ids := []model.ID{
		model.BergetGPTOSS120B,
		model.BergetMistralMedium35,
		model.BergetMistralSmall32,
		model.BergetGLM47,
		model.BergetGLM52,
		model.BergetKimiK26,
		model.BergetGemma431B,
		model.BergetLlama3370B,
	}
	var ok int
	for _, id := range ids {
		c := client(t, id)
		resp, err := c.SendMessages(context.Background(), []message.Message{
			message.NewUserMessage("Reply with one word: hello"),
		}, nil)
		if err != nil {
			t.Logf("FAIL %-50s %v", id, err)
			continue
		}
		ok++
		t.Logf("OK   %-50s -> %q", id, strings.TrimSpace(resp.Content))
	}
	t.Logf("chat models reachable: %d/%d", ok, len(ids))
	if ok == 0 {
		t.Fatal("no chat models reachable")
	}
}
