package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
)

func TestLoop_SingleToolCall(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"hello"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(llmClient, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "done" {
		t.Errorf("expected content 'done', got %q", resp.Content)
	}
	if resp.TotalToolCalls != 1 {
		t.Errorf("expected 1 tool call, got %d", resp.TotalToolCalls)
	}
	if resp.TotalTurns != 2 {
		t.Errorf("expected 2 turns, got %d", resp.TotalTurns)
	}
}

func TestLoop_MultipleToolCallsInOneTurn(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"a"}`,
					Type:  "function",
				},
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
		},
		mockResponse{Content: "all done"},
	)

	a := agent.New(llmClient, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalToolCalls != 3 {
		t.Errorf("expected 3 tool calls, got %d", resp.TotalToolCalls)
	}
	if resp.Content != "all done" {
		t.Errorf("expected content 'all done', got %q", resp.Content)
	}
}

func TestLoop_MultipleIterations(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "echo",
					Input: `{"text":"2"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-3",
					Name:  "echo",
					Input: `{"text":"3"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "final"},
	)

	a := agent.New(llmClient, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalTurns != 4 {
		t.Errorf("expected 4 turns, got %d", resp.TotalTurns)
	}
	if resp.TotalToolCalls != 3 {
		t.Errorf("expected 3 tool calls, got %d", resp.TotalToolCalls)
	}
	if llmClient.CallCount() != 4 {
		t.Errorf("expected 4 LLM calls, got %d", llmClient.CallCount())
	}
	if resp.Content != "final" {
		t.Errorf("expected content 'final', got %q", resp.Content)
	}
}

func TestLoop_EarlyStopNoToolCalls(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{Content: "immediate answer"},
	)

	a := agent.New(llmClient, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalTurns != 1 {
		t.Errorf("expected 1 turn, got %d", resp.TotalTurns)
	}
	if resp.TotalToolCalls != 0 {
		t.Errorf("expected 0 tool calls, got %d", resp.TotalToolCalls)
	}
	if resp.Content != "immediate answer" {
		t.Errorf("expected content 'immediate answer', got %q", resp.Content)
	}
	if llmClient.CallCount() != 1 {
		t.Errorf("expected 1 LLM call, got %d", llmClient.CallCount())
	}
}

func TestLoop_MaxIterations(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "echo",
					Input: `{"text":"2"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-3",
					Name:  "echo",
					Input: `{"text":"3"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-4",
					Name:  "echo",
					Input: `{"text":"4"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "should not reach"},
	)

	a := agent.New(llmClient,
		agent.WithTools(&echoTool{}),
		agent.WithMaxIterations(2),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// iteration 0: call 0 -> tool -> execute -> iteration becomes 1
	// iteration 1: call 1 -> tool -> execute -> iteration becomes 2
	// call 2 -> tool -> iteration(2) >= maxIter(2) -> EXIT with pending tools
	if llmClient.CallCount() != 3 {
		t.Errorf(
			"expected 3 LLM calls with maxIterations=2, got %d",
			llmClient.CallCount(),
		)
	}

	if len(resp.ToolCalls) == 0 {
		t.Error(
			"expected pending tool calls in response when max iterations reached",
		)
	}
}

func TestLoop_MaxTurnsViaOption(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "echo",
					Input: `{"text":"2"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "should not reach"},
	)

	a := agent.New(llmClient, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(context.Background(), "test", agent.WithMaxTurns(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if llmClient.CallCount() != 2 {
		t.Errorf(
			"expected 2 LLM calls with WithMaxTurns(1), got %d",
			llmClient.CallCount(),
		)
	}

	if len(resp.ToolCalls) == 0 {
		t.Error("expected pending tool calls when max turns reached")
	}
}

func TestLoop_ToolError(t *testing.T) {
	var capturedToolError bool
	var mu sync.Mutex

	base := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "error_tool", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "recovered from error"},
	)
	llmClient := &toolResultCapturingLLM{
		base: base,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == message.Tool {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok &&
							tr.IsError {
							mu.Lock()
							capturedToolError = true
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	a := agent.New(llmClient, agent.WithTools(&errorTool{}))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "recovered from error" {
		t.Errorf("expected 'recovered from error', got %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()
	if !capturedToolError {
		t.Error("expected LLM to receive tool result with IsError=true")
	}
}

func TestLoop_ToolIsErrorFlag(t *testing.T) {
	var capturedToolError bool
	var mu sync.Mutex

	base := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "is_error_tool",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "handled the error"},
	)
	llmClient := &toolResultCapturingLLM{
		base: base,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == message.Tool {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok &&
							tr.IsError {
							mu.Lock()
							capturedToolError = true
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	a := agent.New(llmClient, agent.WithTools(&isErrorTool{}))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "handled the error" {
		t.Errorf("expected 'handled the error', got %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()
	if !capturedToolError {
		t.Error("expected LLM to receive tool result with IsError=true")
	}
}

func TestLoop_SequentialToolExecution(t *testing.T) {
	tracker := newConcurrencyTrackingTool(50 * time.Millisecond)

	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "tracking_tool",
					Input: `{"text":"a"}`,
					Type:  "function",
				},
				{
					ID:    "tc-2",
					Name:  "tracking_tool",
					Input: `{"text":"b"}`,
					Type:  "function",
				},
				{
					ID:    "tc-3",
					Name:  "tracking_tool",
					Input: `{"text":"c"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(llmClient,
		agent.WithTools(tracker),
		agent.WithSequentialToolExecution(),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracker.maxConcurrent.Load() != 1 {
		t.Errorf(
			"expected max concurrent 1 for sequential execution, got %d",
			tracker.maxConcurrent.Load(),
		)
	}
}

func TestLoop_ParallelToolExecution(t *testing.T) {
	tracker := newConcurrencyTrackingTool(100 * time.Millisecond)

	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "tracking_tool",
					Input: `{"text":"a"}`,
					Type:  "function",
				},
				{
					ID:    "tc-2",
					Name:  "tracking_tool",
					Input: `{"text":"b"}`,
					Type:  "function",
				},
				{
					ID:    "tc-3",
					Name:  "tracking_tool",
					Input: `{"text":"c"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(llmClient, agent.WithTools(tracker))

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracker.maxConcurrent.Load() <= 1 {
		t.Errorf(
			"expected max concurrent > 1 for parallel execution, got %d",
			tracker.maxConcurrent.Load(),
		)
	}
}

func TestLoop_MaxParallelTools(t *testing.T) {
	tracker := newConcurrencyTrackingTool(100 * time.Millisecond)

	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "tracking_tool",
					Input: `{"text":"a"}`,
					Type:  "function",
				},
				{
					ID:    "tc-2",
					Name:  "tracking_tool",
					Input: `{"text":"b"}`,
					Type:  "function",
				},
				{
					ID:    "tc-3",
					Name:  "tracking_tool",
					Input: `{"text":"c"}`,
					Type:  "function",
				},
				{
					ID:    "tc-4",
					Name:  "tracking_tool",
					Input: `{"text":"d"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(llmClient,
		agent.WithTools(tracker),
		agent.WithMaxParallelTools(2),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracker.maxConcurrent.Load() > 2 {
		t.Errorf(
			"expected max concurrent <= 2 with MaxParallelTools(2), got %d",
			tracker.maxConcurrent.Load(),
		)
	}
}
