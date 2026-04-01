package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
)

func TestMetrics_SingleTurn(t *testing.T) {
	mockLLM := newMockLLM(mockResponse{
		Content: "hello",
		Usage:   llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
	})
	a := agent.New(mockLLM)

	resp, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalTurns != 1 {
		t.Errorf("expected TotalTurns=1, got %d", resp.TotalTurns)
	}
	if resp.TotalToolCalls != 0 {
		t.Errorf("expected TotalToolCalls=0, got %d", resp.TotalToolCalls)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected InputTokens=10, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("expected OutputTokens=5, got %d", resp.Usage.OutputTokens)
	}
	if resp.TotalDuration <= 0 {
		t.Error("expected TotalDuration > 0")
	}
}

func TestMetrics_MultiTurn(t *testing.T) {
	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"a"}`,
					Type:  "function",
				},
			},
			Usage: llm.TokenUsage{InputTokens: 100, OutputTokens: 20},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "echo",
					Input: `{"text":"b"}`,
					Type:  "function",
				},
				{
					ID:    "tc-3",
					Name:  "echo",
					Input: `{"text":"c"}`,
					Type:  "function",
				},
			},
			Usage: llm.TokenUsage{InputTokens: 200, OutputTokens: 30},
		},
		mockResponse{
			Content: "done",
			Usage:   llm.TokenUsage{InputTokens: 300, OutputTokens: 40},
		},
	)

	a := agent.New(mockLLM, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(context.Background(), "test multi-turn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalTurns != 3 {
		t.Errorf("expected TotalTurns=3, got %d", resp.TotalTurns)
	}
	if resp.TotalToolCalls != 3 {
		t.Errorf("expected TotalToolCalls=3 (1+2), got %d", resp.TotalToolCalls)
	}
	if resp.Usage.InputTokens != 600 {
		t.Errorf(
			"expected aggregated InputTokens=600, got %d",
			resp.Usage.InputTokens,
		)
	}
	if resp.Usage.OutputTokens != 90 {
		t.Errorf(
			"expected aggregated OutputTokens=90, got %d",
			resp.Usage.OutputTokens,
		)
	}
}

func TestMetrics_Duration(t *testing.T) {
	mockLLM := newMockLLM(mockResponse{Content: "fast"})
	a := agent.New(mockLLM)

	resp, err := a.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalDuration <= 0 {
		t.Error("expected TotalDuration > 0")
	}
}

func TestMetrics_Stream(t *testing.T) {
	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"a"}`,
					Type:  "function",
				},
			},
			Usage: llm.TokenUsage{
				InputTokens:     50,
				OutputTokens:    10,
				CacheReadTokens: 5,
			},
		},
		mockResponse{
			Content: "stream done",
			Usage: llm.TokenUsage{
				InputTokens:     80,
				OutputTokens:    15,
				CacheReadTokens: 3,
			},
		},
	)

	a := agent.New(mockLLM, agent.WithTools(&echoTool{}))

	var resp *agent.ChatResponse
	for event := range a.ChatStream(context.Background(), "stream test") {
		if event.Type == types.EventComplete && event.Response != nil {
			resp = event.Response
		}
	}

	if resp == nil {
		t.Fatal("expected a ChatResponse from stream")
		return
	}
	if resp.TotalTurns != 2 {
		t.Errorf("expected TotalTurns=2, got %d", resp.TotalTurns)
	}
	if resp.TotalToolCalls != 1 {
		t.Errorf("expected TotalToolCalls=1, got %d", resp.TotalToolCalls)
	}
	if resp.Usage.InputTokens != 130 {
		t.Errorf(
			"expected aggregated InputTokens=130, got %d",
			resp.Usage.InputTokens,
		)
	}
	if resp.Usage.OutputTokens != 25 {
		t.Errorf(
			"expected aggregated OutputTokens=25, got %d",
			resp.Usage.OutputTokens,
		)
	}
	if resp.Usage.CacheReadTokens != 8 {
		t.Errorf(
			"expected aggregated CacheReadTokens=8, got %d",
			resp.Usage.CacheReadTokens,
		)
	}
	if resp.TotalDuration <= 0 {
		t.Error("expected TotalDuration > 0")
	}
}
