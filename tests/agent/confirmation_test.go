package agent

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type confirmableTool struct {
	name     string
	executed bool
}

func (t *confirmableTool) Info() tool.Info {
	info := tool.NewInfo(t.name, "A tool that requires confirmation", struct {
		Text string `json:"text" desc:"Input text" required:"false"`
	}{})
	info.RequireConfirmation = true
	return info
}

func (t *confirmableTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	t.executed = true
	return tool.NewTextResponse("executed: " + params.Input), nil
}

func TestConfirmation_Approved(t *testing.T) {
	ct := &confirmableTool{name: "dangerous"}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "dangerous",
					Input: `{"text":"go"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(
		mock,
		agent.WithTools(ct),
		agent.WithConfirmationProvider(
			func(_ context.Context, _ tool.ConfirmationRequest) (bool, error) {
				return true, nil
			},
		),
	)

	resp, err := a.Chat(context.Background(), "do it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ct.executed {
		t.Fatal("tool should have been executed when confirmation approved")
	}
	if resp.Content != "done" {
		t.Fatalf("expected 'done', got %q", resp.Content)
	}
}

func TestConfirmation_Rejected(t *testing.T) {
	ct := &confirmableTool{name: "dangerous"}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "dangerous",
					Input: `{"text":"go"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "rejected"},
	)

	a := agent.New(
		mock,
		agent.WithTools(ct),
		agent.WithConfirmationProvider(
			func(_ context.Context, _ tool.ConfirmationRequest) (bool, error) {
				return false, nil
			},
		),
	)

	resp, err := a.Chat(context.Background(), "do it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.executed {
		t.Fatal("tool should not have been executed when confirmation rejected")
	}
	if resp.Content != "rejected" {
		t.Fatalf("expected 'rejected', got %q", resp.Content)
	}
}

func TestConfirmation_NoProvider(t *testing.T) {
	ct := &confirmableTool{name: "dangerous"}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "dangerous",
					Input: `{"text":"go"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock, agent.WithTools(ct))

	resp, err := a.Chat(context.Background(), "do it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ct.executed {
		t.Fatal("tool should execute normally when no provider is configured")
	}
	if resp.Content != "done" {
		t.Fatalf("expected 'done', got %q", resp.Content)
	}
}

func TestConfirmation_ProviderReceivesRequest(t *testing.T) {
	ct := &confirmableTool{name: "delete_db"}

	var capturedReq tool.ConfirmationRequest
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-42",
					Name:  "delete_db",
					Input: `{"text":"all"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(
		mock,
		agent.WithTools(ct),
		agent.WithConfirmationProvider(
			func(_ context.Context, req tool.ConfirmationRequest) (bool, error) {
				capturedReq = req
				return true, nil
			},
		),
	)

	_, err := a.Chat(context.Background(), "delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedReq.ToolCallID != "tc-42" {
		t.Errorf("expected ToolCallID 'tc-42', got %q", capturedReq.ToolCallID)
	}
	if capturedReq.ToolName != "delete_db" {
		t.Errorf("expected ToolName 'delete_db', got %q", capturedReq.ToolName)
	}
	if capturedReq.Input != `{"text":"all"}` {
		t.Errorf("expected input, got %q", capturedReq.Input)
	}
}

func TestConfirmation_RequestFromWithinTool(t *testing.T) {
	inToolTool := &simpleTool{
		name: "risky",
		run: func(ctx context.Context, _ tool.Call) (tool.Response, error) {
			if err := tool.RequestConfirmation(ctx, "about to do something risky", map[string]string{"action": "delete"}); err != nil {
				return tool.Response{}, err
			}
			return tool.NewTextResponse("proceeded"), nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "risky", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	var capturedReq tool.ConfirmationRequest
	a := agent.New(
		mock,
		agent.WithTools(inToolTool),
		agent.WithConfirmationProvider(
			func(_ context.Context, req tool.ConfirmationRequest) (bool, error) {
				capturedReq = req
				return true, nil
			},
		),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Fatalf("expected 'done', got %q", resp.Content)
	}
	if capturedReq.Hint != "about to do something risky" {
		t.Errorf("expected hint passed through, got %q", capturedReq.Hint)
	}
	if capturedReq.ToolName != "risky" {
		t.Errorf("expected ToolName 'risky', got %q", capturedReq.ToolName)
	}
}

func TestConfirmation_RequestFromWithinTool_Rejected(t *testing.T) {
	inToolTool := &simpleTool{
		name: "risky",
		run: func(ctx context.Context, _ tool.Call) (tool.Response, error) {
			if err := tool.RequestConfirmation(ctx, "risky op", nil); err != nil {
				return tool.Response{}, err
			}
			return tool.NewTextResponse("should not reach"), nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "risky", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "handled rejection"},
	)

	a := agent.New(
		mock,
		agent.WithTools(inToolTool),
		agent.WithConfirmationProvider(
			func(_ context.Context, _ tool.ConfirmationRequest) (bool, error) {
				return false, nil
			},
		),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "handled rejection" {
		t.Fatalf("expected 'handled rejection', got %q", resp.Content)
	}
}

func TestConfirmation_StreamEmitsEvent(t *testing.T) {
	ct := &confirmableTool{name: "dangerous"}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "dangerous",
					Input: `{"text":"go"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(
		mock,
		agent.WithTools(ct),
		agent.WithConfirmationProvider(
			func(_ context.Context, _ tool.ConfirmationRequest) (bool, error) {
				return true, nil
			},
		),
	)

	var confirmationEvents []agent.ChatEvent
	var finalContent string
	for event := range a.ChatStream(context.Background(), "do it") {
		if event.Type == types.EventConfirmationRequired {
			confirmationEvents = append(confirmationEvents, event)
		}
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if len(confirmationEvents) != 1 {
		t.Fatalf(
			"expected 1 confirmation event, got %d",
			len(confirmationEvents),
		)
	}
	if confirmationEvents[0].ConfirmationRequest == nil {
		t.Fatal("confirmation event should have ConfirmationRequest set")
	}
	if confirmationEvents[0].ConfirmationRequest.ToolName != "dangerous" {
		t.Errorf(
			"expected ToolName 'dangerous', got %q",
			confirmationEvents[0].ConfirmationRequest.ToolName,
		)
	}
	if finalContent != "done" {
		t.Fatalf("expected 'done', got %q", finalContent)
	}
}

func TestConfirmation_WithHookDeny(t *testing.T) {
	ct := &confirmableTool{name: "dangerous"}

	providerCalled := false
	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
			if tc.ToolName == "dangerous" {
				return agent.PreToolUseResult{
					Action:     agent.HookDeny,
					DenyReason: "policy violation",
				}, nil
			}
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "dangerous", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "denied"},
	)

	a := agent.New(
		mock,
		agent.WithTools(ct),
		agent.WithHooks(hooks),
		agent.WithConfirmationProvider(
			func(_ context.Context, _ tool.ConfirmationRequest) (bool, error) {
				providerCalled = true
				return true, nil
			},
		),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.executed {
		t.Fatal("tool should not execute when hook denies")
	}
	if providerCalled {
		t.Fatal(
			"confirmation provider should not be called when hook denies first",
		)
	}
}

func TestConfirmation_ParallelTools(t *testing.T) {
	ct := &confirmableTool{name: "needs_confirm"}
	normalTool := &simpleTool{
		name: "normal",
		run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
			return tool.NewTextResponse("normal result"), nil
		},
	}

	var mu sync.Mutex
	var confirmedTools []string

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "needs_confirm",
					Input: `{"text":"go"}`,
					Type:  "function",
				},
				{ID: "tc-2", Name: "normal", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(
		mock,
		agent.WithTools(ct, normalTool),
		agent.WithConfirmationProvider(
			func(_ context.Context, req tool.ConfirmationRequest) (bool, error) {
				mu.Lock()
				confirmedTools = append(confirmedTools, req.ToolName)
				mu.Unlock()
				return true, nil
			},
		),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ct.executed {
		t.Fatal("confirmable tool should have executed after approval")
	}
	if resp.Content != "done" {
		t.Fatalf("expected 'done', got %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(confirmedTools) != 1 || confirmedTools[0] != "needs_confirm" {
		t.Fatalf(
			"expected only 'needs_confirm' to be confirmed, got %v",
			confirmedTools,
		)
	}
}

func TestConfirmation_Handoff(t *testing.T) {
	ct := &confirmableTool{name: "target_tool"}

	var providerCalledBy string

	targetLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "target_tool",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "target done"},
	)
	targetAgent := agent.New(
		targetLLM,
		agent.WithTools(ct),
		agent.WithConfirmationProvider(
			func(_ context.Context, req tool.ConfirmationRequest) (bool, error) {
				providerCalledBy = req.ToolName
				return true, nil
			},
		),
	)

	sourceLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "transfer_to_Target",
					Input: `{"reason":"needs specialist"}`,
					Type:  "function",
				},
			},
		},
	)
	sourceAgent := agent.New(sourceLLM,
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "Target",
			Description: "Specialist agent",
			Agent:       targetAgent,
		}),
	)

	resp, err := sourceAgent.Chat(context.Background(), "delegate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ct.executed {
		t.Fatal("target agent's tool should have executed after handoff")
	}
	if providerCalledBy != "target_tool" {
		t.Fatalf(
			"expected target agent's provider to be called for 'target_tool', got %q",
			providerCalledBy,
		)
	}
	if resp.Content != "target done" {
		t.Fatalf("expected 'target done', got %q", resp.Content)
	}
}

func TestConfirmation_ProviderError(t *testing.T) {
	ct := &confirmableTool{name: "dangerous"}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "dangerous", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "error handled"},
	)

	a := agent.New(
		mock,
		agent.WithTools(ct),
		agent.WithConfirmationProvider(
			func(_ context.Context, _ tool.ConfirmationRequest) (bool, error) {
				return false, errors.New("provider crashed")
			},
		),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.executed {
		t.Fatal("tool should not execute when provider returns error")
	}
	if resp.Content != "error handled" {
		t.Fatalf("expected 'error handled', got %q", resp.Content)
	}
}

func TestConfirmation_RequestFromWithinTool_NoProvider(t *testing.T) {
	executed := false
	inToolTool := &simpleTool{
		name: "risky",
		run: func(ctx context.Context, _ tool.Call) (tool.Response, error) {
			if err := tool.RequestConfirmation(ctx, "risky op", nil); err != nil {
				return tool.Response{}, err
			}
			executed = true
			return tool.NewTextResponse("done"), nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "risky", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock, agent.WithTools(inToolTool))

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executed {
		t.Fatal(
			"tool should auto-approve RequestConfirmation when no provider configured",
		)
	}
}
