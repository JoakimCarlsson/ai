package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
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

func TestOnModelError_StreamRecovery(t *testing.T) {
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			return agent.ModelErrorResult{
				Action: agent.HookModify,
				Response: &llm.Response{
					Content: "stream recovered",
				},
			}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{Err: fmt.Errorf("stream failed")},
	)

	a := agent.New(mock, agent.WithHooks(hooks))

	var finalContent string
	var gotError bool
	for event := range a.ChatStream(
		context.Background(),
		"test",
	) {
		if event.Type == types.EventComplete &&
			event.Response != nil {
			finalContent = event.Response.Content
		}
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if gotError {
		t.Fatal("should not get error event when recovery succeeds")
	}
	if finalContent != "stream recovered" {
		t.Fatalf(
			"expected 'stream recovered', got %q",
			finalContent,
		)
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

func TestBeforeAgent_StreamShortCircuit(t *testing.T) {
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{
				Action: agent.HookModify,
				Response: &agent.ChatResponse{
					Content: "stream short-circuited",
				},
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "should not reach"})
	a := agent.New(mock, agent.WithHooks(hooks))

	var finalContent string
	for event := range a.ChatStream(
		context.Background(),
		"test",
	) {
		if event.Type == types.EventComplete &&
			event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "stream short-circuited" {
		t.Fatalf(
			"expected 'stream short-circuited', got %q",
			finalContent,
		)
	}
	if mock.CallCount() != 0 {
		t.Fatal("LLM should not have been called in stream short-circuit")
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

func TestAfterAgent_StreamModifyResponse(t *testing.T) {
	hooks := agent.Hooks{
		AfterAgent: func(_ context.Context, ac agent.LifecycleContext) (agent.LifecycleResult, error) {
			modified := *ac.Response
			modified.Content = "stream modified: " + modified.Content
			return agent.LifecycleResult{
				Action:   agent.HookModify,
				Response: &modified,
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "original"})
	a := agent.New(mock, agent.WithHooks(hooks))

	var finalContent string
	for event := range a.ChatStream(
		context.Background(),
		"test",
	) {
		if event.Type == types.EventComplete &&
			event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "stream modified: original" {
		t.Fatalf(
			"expected 'stream modified: original', got %q",
			finalContent,
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

func TestOnUserMessage_StreamDeny(t *testing.T) {
	hooks := agent.Hooks{
		OnUserMessage: func(_ context.Context, _ agent.UserMessageContext) (agent.UserMessageResult, error) {
			return agent.UserMessageResult{
				Action:     agent.HookDeny,
				DenyReason: "blocked",
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "should not reach"})
	a := agent.New(mock, agent.WithHooks(hooks))

	var gotError bool
	for event := range a.ChatStream(
		context.Background(),
		"test",
	) {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Fatal("expected error event when user message is denied in stream")
	}
	if mock.CallCount() != 0 {
		t.Fatal(
			"LLM should not be called when user message is denied in stream",
		)
	}
}

func TestOnEvent_FiresForAllHookTypes(t *testing.T) {
	var mu sync.Mutex
	var eventTypes []agent.HookEventType

	hooks := agent.Hooks{
		OnEvent: func(_ context.Context, evt agent.HookEvent) {
			mu.Lock()
			eventTypes = append(eventTypes, evt.Type)
			mu.Unlock()
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"hi"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(&echoTool{}),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	typesSeen := make(map[agent.HookEventType]bool)
	for _, et := range eventTypes {
		typesSeen[et] = true
	}

	expected := []agent.HookEventType{
		agent.HookEventBeforeRun,
		agent.HookEventUserMessage,
		agent.HookEventBeforeAgent,
		agent.HookEventPreModelCall,
		agent.HookEventPostModelCall,
		agent.HookEventPreToolUse,
		agent.HookEventPostToolUse,
		agent.HookEventAfterAgent,
		agent.HookEventAfterRun,
	}
	for _, et := range expected {
		if !typesSeen[et] {
			t.Errorf("expected OnEvent to fire for %q", et)
		}
	}
}

func TestOnEvent_ToolError(t *testing.T) {
	var mu sync.Mutex
	var eventTypes []agent.HookEventType

	hooks := agent.Hooks{
		OnEvent: func(_ context.Context, evt agent.HookEvent) {
			mu.Lock()
			eventTypes = append(eventTypes, evt.Type)
			mu.Unlock()
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "error_tool",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(&errorTool{}),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	typesSeen := make(map[agent.HookEventType]bool)
	for _, et := range eventTypes {
		typesSeen[et] = true
	}

	if !typesSeen[agent.HookEventToolError] {
		t.Error("expected OnEvent to fire for tool_error")
	}
}

func TestNewObservingHooks_IncludesNewHookTypes(t *testing.T) {
	collector := &hookEventCollector{}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "error_tool",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(&errorTool{}),
		agent.WithHooks(
			agent.NewObservingHooks(collector.collect),
		),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := collector.all()
	typesSeen := make(map[agent.HookEventType]bool)
	for _, evt := range events {
		typesSeen[evt.Type] = true
	}

	expected := []agent.HookEventType{
		agent.HookEventBeforeRun,
		agent.HookEventAfterRun,
		agent.HookEventBeforeAgent,
		agent.HookEventAfterAgent,
		agent.HookEventUserMessage,
		agent.HookEventToolError,
	}
	for _, et := range expected {
		if !typesSeen[et] {
			t.Errorf(
				"expected NewObservingHooks to emit %q event",
				et,
			)
		}
	}
}
