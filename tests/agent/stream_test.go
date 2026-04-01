package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/codes"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
)

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

func TestBeforeAgent_StreamHookError(t *testing.T) {
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{}, fmt.Errorf("before-agent failed")
		},
	}

	mock := newMockLLM(mockResponse{Content: "unreachable"})
	a := agent.New(mock, agent.WithHooks(hooks))

	var gotError bool
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Fatal("expected error event from BeforeAgent hook in stream")
	}
	if mock.CallCount() != 0 {
		t.Fatal("LLM should not be called when BeforeAgent errors in stream")
	}
}

func TestAfterAgent_StreamHookError(t *testing.T) {
	hooks := agent.Hooks{
		AfterAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			return agent.LifecycleResult{}, fmt.Errorf("after-agent failed")
		},
	}

	mock := newMockLLM(mockResponse{Content: "original"})
	a := agent.New(mock, agent.WithHooks(hooks))

	var gotError bool
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Fatal("expected error event from AfterAgent hook in stream")
	}
}

func TestOnUserMessage_StreamHookError(t *testing.T) {
	hooks := agent.Hooks{
		OnUserMessage: func(_ context.Context, _ agent.UserMessageContext) (agent.UserMessageResult, error) {
			return agent.UserMessageResult{}, fmt.Errorf("validation failed")
		},
	}

	mock := newMockLLM(mockResponse{Content: "unreachable"})
	a := agent.New(mock, agent.WithHooks(hooks))

	var gotError bool
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Fatal("expected error event from OnUserMessage hook in stream")
	}
}

func TestBeforeRun_AfterRun_StreamFire(t *testing.T) {
	var beforeFired, afterFired bool
	hooks := agent.Hooks{
		BeforeRun: func(_ context.Context, _ agent.RunContext) {
			beforeFired = true
		},
		AfterRun: func(_ context.Context, _ agent.RunContext) {
			afterFired = true
		},
	}

	mock := newMockLLM(mockResponse{Content: "hello"})
	a := agent.New(mock, agent.WithHooks(hooks))

	for event := range a.ChatStream(context.Background(), "test") {
		_ = event
	}

	if !beforeFired {
		t.Fatal("BeforeRun should fire in ChatStream")
	}
	if !afterFired {
		t.Fatal("AfterRun should fire in ChatStream")
	}
}

func TestAfterRun_StreamOnError(t *testing.T) {
	var afterError error
	hooks := agent.Hooks{
		AfterRun: func(_ context.Context, rc agent.RunContext) {
			afterError = rc.Error
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("boom")})
	a := agent.New(mock, agent.WithHooks(hooks))

	for event := range a.ChatStream(context.Background(), "test") {
		_ = event
	}

	if afterError == nil {
		t.Fatal("AfterRun should receive the error in stream")
	}
}

func TestOnUserMessage_StreamModify(t *testing.T) {
	hooks := agent.Hooks{
		OnUserMessage: func(_ context.Context, uc agent.UserMessageContext) (agent.UserMessageResult, error) {
			return agent.UserMessageResult{
				Action:  agent.HookModify,
				Message: "modified: " + uc.Message,
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "done"})
	a := agent.New(mock, agent.WithHooks(hooks))

	var finalContent string
	for event := range a.ChatStream(context.Background(), "original") {
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "done" {
		t.Fatalf("expected 'done', got %q", finalContent)
	}
}

func TestOnModelError_StreamNoRecovery(t *testing.T) {
	var hookFired bool
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			hookFired = true
			return agent.ModelErrorResult{Action: agent.HookAllow}, nil
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("stream failed")})
	a := agent.New(mock, agent.WithHooks(hooks))

	var gotError bool
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !hookFired {
		t.Fatal("OnModelError should fire in stream")
	}
	if !gotError {
		t.Fatal("expected error event when no recovery")
	}
}

func TestOnModelError_StreamHookReturnsError(t *testing.T) {
	hooks := agent.Hooks{
		OnModelError: func(_ context.Context, _ agent.ModelErrorContext) (agent.ModelErrorResult, error) {
			return agent.ModelErrorResult{}, fmt.Errorf("hook exploded")
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("stream failed")})
	a := agent.New(mock, agent.WithHooks(hooks))

	var gotError bool
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Fatal("expected error event when hook also errors")
	}
}

func TestOnToolError_StreamRecovery(t *testing.T) {
	hooks := agent.Hooks{
		OnToolError: func(_ context.Context, tc agent.ToolErrorContext) (agent.ToolErrorResult, error) {
			return agent.ToolErrorResult{
				Action: agent.HookModify,
				Output: "recovered: " + tc.Output,
			}, nil
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

	var finalContent string
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "done" {
		t.Fatalf("expected 'done', got %q", finalContent)
	}
}

func TestBeforeAgent_Deny_StreamWithAfterHooks(t *testing.T) {
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

	for event := range a.ChatStream(context.Background(), "test") {
		_ = event
	}

	if !afterAgentFired {
		t.Fatal("AfterAgent should fire even when BeforeAgent denies in stream")
	}
	if !afterRunFired {
		t.Fatal("AfterRun should fire even when BeforeAgent denies in stream")
	}
}

func TestBeforeRun_AfterRun_StreamFire_WithTools(t *testing.T) {
	var afterDuration agent.RunContext
	hooks := agent.Hooks{
		AfterRun: func(_ context.Context, rc agent.RunContext) {
			afterDuration = rc
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

	for event := range a.ChatStream(context.Background(), "test") {
		_ = event
	}

	if afterDuration.Duration == 0 {
		t.Fatal("AfterRun should have non-zero duration")
	}
	if afterDuration.Response == nil {
		t.Fatal("AfterRun should have response")
	}
}

func TestOnEvent_ModelError_Stream(t *testing.T) {
	var mu sync.Mutex
	var eventTypes []agent.HookEventType

	hooks := agent.Hooks{
		OnEvent: func(_ context.Context, evt agent.HookEvent) {
			mu.Lock()
			eventTypes = append(eventTypes, evt.Type)
			mu.Unlock()
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("boom")})
	a := agent.New(mock, agent.WithHooks(hooks))

	for event := range a.ChatStream(context.Background(), "test") {
		_ = event
	}

	mu.Lock()
	defer mu.Unlock()

	typesSeen := make(map[agent.HookEventType]bool)
	for _, et := range eventTypes {
		typesSeen[et] = true
	}

	if !typesSeen[agent.HookEventModelError] {
		t.Error("expected OnEvent to fire for model_error in stream")
	}
}

func TestChatStream_CreatesSpans(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc1",
					Name:  "echo",
					Input: `{"text":"hi"}`,
				},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock, agent.WithTools(&echoTool{}))
	for evt := range a.ChatStream(
		context.Background(),
		"test",
	) {
		_ = evt
	}

	spans := exporter.GetSpans()
	if findSpan(spans, "invoke_agent") == nil {
		t.Fatal("expected invoke_agent span")
	}
	if findSpan(spans, "execute_tool") == nil {
		t.Fatal("expected execute_tool span")
	}
}

func TestChatStream_HookError_RecordsOnSpan(t *testing.T) {
	exporter := setupTracing(t)

	mock := newMockLLM(mockResponse{Content: "hi"})
	a := agent.New(mock,
		agent.WithHooks(agent.Hooks{
			OnUserMessage: func(
				_ context.Context,
				_ agent.UserMessageContext,
			) (agent.UserMessageResult, error) {
				return agent.UserMessageResult{},
					fmt.Errorf("hook failed")
			},
		}),
	)

	for evt := range a.ChatStream(
		context.Background(),
		"test",
	) {
		_ = evt
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span == nil {
		t.Fatal("expected invoke_agent span")
		return
	}
	if span.Status.Code != codes.Error {
		t.Error("expected error status on span from hook error")
	}
}
