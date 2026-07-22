package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
)

func TestOption_WithContinuationProvider(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"1"}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "echo", Input: `{"text":"2"}`, Type: "function"},
			},
		},
		mockResponse{Content: "finished"},
	)

	called := false
	provider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationDecision, error) {
		called = true
		return agent.ContinuationApprove, nil
	}

	a := agent.New(llmClient,
		agent.WithTools(&echoTool{}),
		agent.WithMaxIterations(1),
		agent.WithContinuationProvider(provider),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected continuation provider to be called")
	}
}

func TestLoop_Continuation_Approve(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"1"}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "echo", Input: `{"text":"2"}`, Type: "function"},
			},
		},
		mockResponse{
			Content:      "finished after continuation",
			FinishReason: message.FinishReasonEndTurn,
		},
	)

	var capturedReq agent.ContinuationRequest
	provider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationDecision, error) {
		capturedReq = req
		return agent.ContinuationApprove, nil
	}

	a := agent.New(llmClient,
		agent.WithTools(&echoTool{}),
		agent.WithMaxIterations(1),
		agent.WithContinuationProvider(provider),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "finished after continuation" {
		t.Errorf("expected finished content, got %q", resp.Content)
	}
	if resp.FinishReason != message.FinishReasonEndTurn {
		t.Errorf("expected FinishReason %q, got %q", message.FinishReasonEndTurn, resp.FinishReason)
	}
	if capturedReq.MaxIterations != 1 {
		t.Errorf("expected MaxIterations 1, got %d", capturedReq.MaxIterations)
	}
	if capturedReq.TotalIterations != 1 {
		t.Errorf("expected TotalIterations 1, got %d", capturedReq.TotalIterations)
	}
}

func TestLoop_Continuation_Decline(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"1"}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "echo", Input: `{"text":"2"}`, Type: "function"},
			},
		},
		mockResponse{Content: "summarized after decline"},
	)

	provider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationDecision, error) {
		return agent.ContinuationDecline, nil
	}

	a := agent.New(llmClient,
		agent.WithTools(&echoTool{}),
		agent.WithMaxIterations(1),
		agent.WithContinuationProvider(provider),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "summarized after decline" {
		t.Errorf("expected summarized content, got %q", resp.Content)
	}
	if resp.FinishReason != message.FinishReasonMaxIterations {
		t.Errorf("expected FinishReasonMaxIterations, got %q", resp.FinishReason)
	}
}

func TestLoop_Continuation_Timeout(t *testing.T) {
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"1"}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "echo", Input: `{"text":"2"}`, Type: "function"},
			},
		},
		mockResponse{Content: "summarized after timeout"},
	)

	provider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationDecision, error) {
		return agent.ContinuationTimeout, nil
	}

	a := agent.New(llmClient,
		agent.WithTools(&echoTool{}),
		agent.WithMaxIterations(1),
		agent.WithContinuationProvider(provider),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "summarized after timeout" {
		t.Errorf("expected summarized content, got %q", resp.Content)
	}
}
