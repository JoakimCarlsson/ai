package agent

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel/codes"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
)

func TestOnToolError_Recovery(t *testing.T) {
	hooks := agent.Hooks{
		OnToolError: func(_ context.Context, tc agent.ToolErrorContext) (agent.ToolErrorResult, error) {
			return agent.ToolErrorResult{
				Action: agent.HookModify,
				Output: "recovered from: " + tc.Output,
			}, nil
		},
	}

	var capturedToolResult string
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "error_tool", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							capturedToolResult = tr.Content
						}
					}
				}
			}
		},
	}

	a := agent.New(parentLLM,
		agent.WithTools(&errorTool{}),
		agent.WithHooks(hooks),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Fatalf("expected 'done', got %q", resp.Content)
	}
	if capturedToolResult != "recovered from: tool failed" {
		t.Fatalf(
			"expected recovered output, got %q",
			capturedToolResult,
		)
	}
}

func TestOnToolError_NoRecovery(t *testing.T) {
	var hookFired bool
	hooks := agent.Hooks{
		OnToolError: func(_ context.Context, _ agent.ToolErrorContext) (agent.ToolErrorResult, error) {
			hookFired = true
			return agent.ToolErrorResult{Action: agent.HookAllow}, nil
		},
	}

	var capturedIsError bool
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "error_tool", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							capturedIsError = tr.IsError
						}
					}
				}
			}
		},
	}

	a := agent.New(parentLLM,
		agent.WithTools(&errorTool{}),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hookFired {
		t.Fatal("OnToolError hook should have fired")
	}
	if !capturedIsError {
		t.Fatal(
			"tool result should still be error when hook returns Allow",
		)
	}
}

func TestOnToolError_FirstRecoveryWins(t *testing.T) {
	hook1 := agent.Hooks{
		OnToolError: func(_ context.Context, _ agent.ToolErrorContext) (agent.ToolErrorResult, error) {
			return agent.ToolErrorResult{
				Action: agent.HookModify,
				Output: "first recovery",
			}, nil
		},
	}
	hook2 := agent.Hooks{
		OnToolError: func(_ context.Context, _ agent.ToolErrorContext) (agent.ToolErrorResult, error) {
			return agent.ToolErrorResult{
				Action: agent.HookModify,
				Output: "second recovery",
			}, nil
		},
	}

	var capturedToolResult string
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "error_tool", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							capturedToolResult = tr.Content
						}
					}
				}
			}
		},
	}

	a := agent.New(parentLLM,
		agent.WithTools(&errorTool{}),
		agent.WithHooks(hook1, hook2),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedToolResult != "first recovery" {
		t.Fatalf(
			"expected first recovery to win, got %q",
			capturedToolResult,
		)
	}
}

func TestOnToolError_IsErrorTool(t *testing.T) {
	var hookFired bool
	hooks := agent.Hooks{
		OnToolError: func(_ context.Context, tc agent.ToolErrorContext) (agent.ToolErrorResult, error) {
			hookFired = true
			return agent.ToolErrorResult{
				Action: agent.HookModify,
				Output: "fixed: " + tc.Output,
			}, nil
		},
	}

	mock := newMockLLM(
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
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(&isErrorTool{}),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hookFired {
		t.Fatal(
			"OnToolError should fire for tools returning IsError=true",
		)
	}
}

func TestOnModelError_Recovery(t *testing.T) {
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			return agent.ModelErrorResult{
				Action: agent.HookModify,
				Response: &llm.Response{
					Content: "recovered response",
				},
			}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{Err: fmt.Errorf("llm exploded")},
	)

	a := agent.New(mock, agent.WithHooks(hooks))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("expected recovery, got error: %v", err)
	}
	if resp.Content != "recovered response" {
		t.Fatalf(
			"expected 'recovered response', got %q",
			resp.Content,
		)
	}
}

func TestOnModelError_NoRecovery(t *testing.T) {
	var hookFired bool
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			hookFired = true
			return agent.ModelErrorResult{
				Action: agent.HookAllow,
			}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{Err: fmt.Errorf("llm exploded")},
	)

	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when hook does not recover")
	}
	if !hookFired {
		t.Fatal("OnModelError hook should have fired")
	}
}

func TestBeforeAgent_ShortCircuit(t *testing.T) {
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{
				Action: agent.HookModify,
				Response: &agent.ChatResponse{
					Content: "short-circuited",
				},
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "should not reach"})
	a := agent.New(mock, agent.WithHooks(hooks))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "short-circuited" {
		t.Fatalf(
			"expected 'short-circuited', got %q",
			resp.Content,
		)
	}
	if mock.CallCount() != 0 {
		t.Fatal(
			"LLM should not have been called when BeforeAgent short-circuits",
		)
	}
}

func TestBeforeAgent_Deny(t *testing.T) {
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{
				Action: agent.HookDeny,
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "should not reach"})
	a := agent.New(mock, agent.WithHooks(hooks))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil && resp.Content != "" {
		t.Fatalf("expected nil/empty response on deny, got %q", resp.Content)
	}
	if mock.CallCount() != 0 {
		t.Fatal("LLM should not have been called when BeforeAgent denies")
	}
}

func TestAfterAgent_ModifyResponse(t *testing.T) {
	hooks := agent.Hooks{
		AfterAgent: func(_ context.Context, ac agent.LifecycleContext) (agent.LifecycleResult, error) {
			modified := *ac.Response
			modified.Content = "modified: " + modified.Content
			return agent.LifecycleResult{
				Action:   agent.HookModify,
				Response: &modified,
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "original"})
	a := agent.New(mock, agent.WithHooks(hooks))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "modified: original" {
		t.Fatalf(
			"expected 'modified: original', got %q",
			resp.Content,
		)
	}
}

func TestBeforeRun_AfterRun_Fire(t *testing.T) {
	var beforeFired, afterFired bool
	var afterInput string
	var afterResp *agent.ChatResponse

	hooks := agent.Hooks{
		BeforeRun: func(_ context.Context, _ agent.RunContext) {
			beforeFired = true
		},
		AfterRun: func(_ context.Context, rc agent.RunContext) {
			afterFired = true
			afterInput = rc.Input
			afterResp = rc.Response
		},
	}

	mock := newMockLLM(mockResponse{Content: "hello"})
	a := agent.New(mock, agent.WithHooks(hooks))

	resp, err := a.Chat(context.Background(), "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !beforeFired {
		t.Fatal("BeforeRun should have fired")
	}
	if !afterFired {
		t.Fatal("AfterRun should have fired")
	}
	if afterInput != "test input" {
		t.Fatalf(
			"AfterRun should have input 'test input', got %q",
			afterInput,
		)
	}
	if afterResp == nil || afterResp.Content != resp.Content {
		t.Fatal("AfterRun should have the response")
	}
}

func TestBeforeRun_AfterRun_OnError(t *testing.T) {
	var afterError error
	hooks := agent.Hooks{
		AfterRun: func(_ context.Context, rc agent.RunContext) {
			afterError = rc.Error
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("boom")})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, _ = a.Chat(context.Background(), "test")
	if afterError == nil {
		t.Fatal("AfterRun should receive the error")
	}
}

func TestOnUserMessage_Modify(t *testing.T) {
	hooks := agent.Hooks{
		OnUserMessage: func(_ context.Context, uc agent.UserMessageContext) (agent.UserMessageResult, error) {
			return agent.UserMessageResult{
				Action:  agent.HookModify,
				Message: "modified: " + uc.Message,
			}, nil
		},
	}

	var capturedMessages []message.Message
	parentBase := newMockLLM(mockResponse{Content: "done"})
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			capturedMessages = msgs
		},
	}

	a := agent.New(parentLLM, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "original")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, msg := range capturedMessages {
		for _, part := range msg.Parts {
			if ct, ok := part.(message.TextContent); ok {
				if ct.Text == "modified: original" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("expected modified user message in LLM call")
	}
}

func TestOnUserMessage_Deny(t *testing.T) {
	hooks := agent.Hooks{
		OnUserMessage: func(_ context.Context, _ agent.UserMessageContext) (agent.UserMessageResult, error) {
			return agent.UserMessageResult{
				Action:     agent.HookDeny,
				DenyReason: "not allowed",
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "should not reach"})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when user message is denied")
	}
	if mock.CallCount() != 0 {
		t.Fatal("LLM should not be called when user message is denied")
	}
}

func TestOnToolError_HookReturnsError(t *testing.T) {
	hooks := agent.Hooks{
		OnToolError: func(_ context.Context, _ agent.ToolErrorContext) (agent.ToolErrorResult, error) {
			return agent.ToolErrorResult{}, fmt.Errorf("hook exploded")
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "error_tool", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(&errorTool{}),
		agent.WithHooks(hooks),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Fatalf("expected 'done', got %q", resp.Content)
	}
}

func TestOnModelError_HookReturnsError(t *testing.T) {
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			return agent.ModelErrorResult{}, fmt.Errorf("hook exploded")
		},
	}

	mock := newMockLLM(
		mockResponse{Err: fmt.Errorf("llm failed")},
	)

	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when both LLM and hook fail")
	}
}

func TestBeforeAgent_HookReturnsError(t *testing.T) {
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{}, fmt.Errorf("before-agent failed")
		},
	}

	mock := newMockLLM(mockResponse{Content: "unreachable"})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from BeforeAgent hook")
	}
	if mock.CallCount() != 0 {
		t.Fatal("LLM should not be called when BeforeAgent errors")
	}
}

func TestAfterAgent_HookReturnsError(t *testing.T) {
	hooks := agent.Hooks{
		AfterAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{}, fmt.Errorf("after-agent failed")
		},
	}

	mock := newMockLLM(mockResponse{Content: "original"})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from AfterAgent hook")
	}
}

func TestOnUserMessage_HookReturnsError(t *testing.T) {
	hooks := agent.Hooks{
		OnUserMessage: func(_ context.Context, _ agent.UserMessageContext) (agent.UserMessageResult, error) {
			return agent.UserMessageResult{}, fmt.Errorf("validation exploded")
		},
	}

	mock := newMockLLM(mockResponse{Content: "unreachable"})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from OnUserMessage hook")
	}
	if mock.CallCount() != 0 {
		t.Fatal("LLM should not be called when OnUserMessage errors")
	}
}

func TestBeforeAgent_Deny_WithAfterHooks(t *testing.T) {
	var afterAgentFired, afterRunFired bool

	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{Action: agent.HookDeny}, nil
		},
		AfterAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			afterAgentFired = true
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
		},
		AfterRun: func(_ context.Context, _ agent.RunContext) {
			afterRunFired = true
		},
	}

	mock := newMockLLM(mockResponse{Content: "unreachable"})
	a := agent.New(mock, agent.WithHooks(hooks))

	a.Chat(context.Background(), "test")

	if !afterAgentFired {
		t.Fatal("AfterAgent should fire even when BeforeAgent denies")
	}
	if !afterRunFired {
		t.Fatal("AfterRun should fire even when BeforeAgent denies")
	}
}

func TestOnModelError_RecoveryWithToolCalls(t *testing.T) {
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			return agent.ModelErrorResult{
				Action: agent.HookModify,
				Response: &llm.Response{
					Content: "recovered with no tools",
				},
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("llm failed")})
	a := agent.New(mock,
		agent.WithTools(&echoTool{}),
		agent.WithHooks(hooks),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("expected recovery, got error: %v", err)
	}
	if resp.Content != "recovered with no tools" {
		t.Fatalf("expected 'recovered with no tools', got %q", resp.Content)
	}
}

func TestOnModelError_HookReturnsError_Chat(t *testing.T) {
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			return agent.ModelErrorResult{}, fmt.Errorf("hook also failed")
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("llm failed")})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when both LLM and OnModelError hook fail")
	}
}

func TestOnModelError_RecoveryNilResponse(t *testing.T) {
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			return agent.ModelErrorResult{
				Action:   agent.HookModify,
				Response: nil,
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("llm failed")})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when recovery response is nil")
	}
}

func TestChat_CreatesInvokeAgentSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(mockResponse{Content: "hello"})

	a := agent.New(mock)
	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span == nil {
		t.Fatal("expected invoke_agent span")
	}
	if spanAttr(span, "gen_ai.operation.name") != "invoke_agent" {
		t.Errorf(
			"expected operation.name 'invoke_agent', got %q",
			spanAttr(span, "gen_ai.operation.name"),
		)
	}
}

func TestChat_RecordsErrorOnSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{Err: fmt.Errorf("provider error")},
	)

	a := agent.New(mock)
	_, err := a.Chat(context.Background(), "hi")
	if err == nil {
		t.Fatal("expected error")
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span == nil {
		t.Fatal("expected invoke_agent span")
		return
	}
	if span.Status.Code != codes.Error {
		t.Error("expected error status on invoke_agent span")
	}
}

func TestChat_RecordsUsageAttrs(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(mockResponse{
		Content: "done",
		Usage: llm.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	})

	a := agent.New(mock)
	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span == nil {
		t.Fatal("expected invoke_agent span")
	}
	if spanAttrInt(span, "gen_ai.usage.input_tokens") != 100 {
		t.Errorf(
			"expected input_tokens 100, got %d",
			spanAttrInt(span, "gen_ai.usage.input_tokens"),
		)
	}
	if spanAttrInt(span, "gen_ai.usage.output_tokens") != 50 {
		t.Errorf(
			"expected output_tokens 50, got %d",
			spanAttrInt(span, "gen_ai.usage.output_tokens"),
		)
	}
	if spanAttrInt(span, "gen_ai.agent.total_turns") != 1 {
		t.Errorf(
			"expected total_turns 1, got %d",
			spanAttrInt(span, "gen_ai.agent.total_turns"),
		)
	}
}

func TestContinue_RecordsErrorOnSpan(t *testing.T) {
	exporter := setupTracing(t)

	a := agent.New(
		newMockLLM(mockResponse{Content: "hi"}),
	)
	_, err := a.Continue(
		context.Background(),
		[]message.ToolResult{
			{ToolCallID: "tc1", Content: "result"},
		},
	)
	if err == nil {
		t.Fatal("expected error (no session)")
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span != nil && span.Status.Code != codes.Error {
		t.Error("expected error status on invoke_agent span")
	}
}
