package tokens

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
)

func newCounter(t *testing.T) *tokens.Counter {
	t.Helper()
	c, err := tokens.NewCounter()
	if err != nil {
		t.Fatalf("NewCounter error: %v", err)
	}
	return c
}

func TestNewCounter(t *testing.T) {
	c := newCounter(t)
	if c == nil {
		t.Fatal("expected non-nil counter")
	}
}

func TestCountTokens_SystemPrompt(t *testing.T) {
	c := newCounter(t)
	result, err := c.CountTokens(
		context.Background(),
		tokens.CountOptions{SystemPrompt: "You are helpful."},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemTokens <= tokens.SystemMessageOverhead {
		t.Error("expected system tokens > overhead")
	}
	if result.MessageTokens != 0 {
		t.Errorf(
			"expected 0 message tokens, got %d",
			result.MessageTokens,
		)
	}
	if result.TotalTokens != result.SystemTokens {
		t.Errorf(
			"total (%d) should equal system (%d)",
			result.TotalTokens,
			result.SystemTokens,
		)
	}
}

func TestCountTokens_EmptySystemPrompt(t *testing.T) {
	c := newCounter(t)
	result, _ := c.CountTokens(
		context.Background(),
		tokens.CountOptions{},
	)
	if result.SystemTokens != 0 {
		t.Errorf(
			"expected 0 system tokens, got %d",
			result.SystemTokens,
		)
	}
}

func TestCountTokens_Messages(t *testing.T) {
	c := newCounter(t)
	result, err := c.CountTokens(
		context.Background(),
		tokens.CountOptions{
			Messages: []message.Message{
				message.NewUserMessage("Hello world"),
				message.NewUserMessage("How are you?"),
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageTokens <= 2*tokens.MessageOverhead {
		t.Error("expected message tokens > 2x overhead")
	}
	if result.TotalTokens != result.MessageTokens {
		t.Error("total should equal message tokens")
	}
}

func TestCountTokens_SystemRoleSkipped(t *testing.T) {
	c := newCounter(t)
	result, _ := c.CountTokens(
		context.Background(),
		tokens.CountOptions{
			Messages: []message.Message{
				message.NewSystemMessage("system in messages"),
				message.NewUserMessage("user text"),
			},
		},
	)

	resultUserOnly, _ := c.CountTokens(
		context.Background(),
		tokens.CountOptions{
			Messages: []message.Message{
				message.NewUserMessage("user text"),
			},
		},
	)

	if result.MessageTokens != resultUserOnly.MessageTokens {
		t.Errorf(
			"system role msg should be skipped: with=%d without=%d",
			result.MessageTokens,
			resultUserOnly.MessageTokens,
		)
	}
}

func TestCountTokens_ToolCallParts(t *testing.T) {
	c := newCounter(t)
	msg := message.NewMessage(
		message.Assistant,
		[]message.ContentPart{
			message.ToolCall{
				ID:    "tc_1",
				Name:  "search",
				Input: `{"query":"golang"}`,
			},
		},
	)

	result, _ := c.CountTokens(
		context.Background(),
		tokens.CountOptions{Messages: []message.Message{msg}},
	)
	if result.MessageTokens <= tokens.MessageOverhead+tokens.ToolCallOverhead {
		t.Error(
			"expected tokens for tool call name + input + overhead",
		)
	}
}

func TestCountTokens_ToolResultParts(t *testing.T) {
	c := newCounter(t)
	msg := message.NewMessage(
		message.Tool,
		[]message.ContentPart{
			message.ToolResult{
				ToolCallID: "tc_1",
				Content:    "Here are the search results for golang.",
			},
		},
	)

	result, _ := c.CountTokens(
		context.Background(),
		tokens.CountOptions{Messages: []message.Message{msg}},
	)
	if result.MessageTokens <= tokens.MessageOverhead+tokens.ToolResultOverhead {
		t.Error("expected tokens for tool result content + overhead")
	}
}

type stubCounterTool struct{}

func (s *stubCounterTool) Info() tool.Info {
	return tool.NewInfo(
		"weather",
		"Get the current weather for a location",
		struct {
			Location string `json:"location" desc:"City name"`
		}{},
	)
}

func (s *stubCounterTool) Run(
	_ context.Context,
	_ tool.Call,
) (tool.Response, error) {
	return tool.NewTextResponse("sunny"), nil
}

func TestCountTokens_Tools(t *testing.T) {
	c := newCounter(t)
	result, err := c.CountTokens(
		context.Background(),
		tokens.CountOptions{
			Tools: []tool.BaseTool{&stubCounterTool{}},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ToolTokens <= tokens.ToolDefinitionOverhead {
		t.Error("expected tool tokens > overhead")
	}
	if result.TotalTokens != result.ToolTokens {
		t.Error("total should equal tool tokens")
	}
}

func TestCountTokens_TotalBreakdown(t *testing.T) {
	c := newCounter(t)
	result, _ := c.CountTokens(
		context.Background(),
		tokens.CountOptions{
			SystemPrompt: "Be helpful.",
			Messages: []message.Message{
				message.NewUserMessage("Hello"),
			},
			Tools: []tool.BaseTool{&stubCounterTool{}},
		},
	)

	expected := result.SystemTokens +
		result.MessageTokens +
		result.ToolTokens
	if result.TotalTokens != expected {
		t.Errorf(
			"total (%d) != system(%d)+message(%d)+tool(%d)",
			result.TotalTokens,
			result.SystemTokens,
			result.MessageTokens,
			result.ToolTokens,
		)
	}
}

func TestCountTokens_ImageURLDefaultTokens(t *testing.T) {
	c := newCounter(t)
	msg := message.NewMessage(message.User, []message.ContentPart{
		message.ImageURLContent{URL: "http://img.png"},
	})

	result, _ := c.CountTokens(
		context.Background(),
		tokens.CountOptions{Messages: []message.Message{msg}},
	)
	if result.MessageTokens <= tokens.MessageOverhead {
		t.Error("expected extra tokens for image URL")
	}
}

func TestCountTokens_Deterministic(t *testing.T) {
	c := newCounter(t)
	opts := tokens.CountOptions{
		SystemPrompt: "You are a helpful assistant.",
		Messages: []message.Message{
			message.NewUserMessage("What is 2+2?"),
		},
	}

	r1, _ := c.CountTokens(context.Background(), opts)
	r2, _ := c.CountTokens(context.Background(), opts)

	if r1.TotalTokens != r2.TotalTokens {
		t.Errorf(
			"non-deterministic: %d vs %d",
			r1.TotalTokens,
			r2.TotalTokens,
		)
	}
}
