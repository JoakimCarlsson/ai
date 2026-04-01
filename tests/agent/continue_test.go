package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

func TestContinue_Basic(t *testing.T) {
	store := session.MemoryStore()

	mockLLM := newMockLLM(
		// First Chat() call: LLM requests a tool call
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "search",
					Input: `{"q":"weather"}`,
					Type:  "function",
				},
			},
		},
		// Second call (via Continue): LLM produces final answer
		mockResponse{Content: "The weather is sunny."},
	)

	a := agent.New(mockLLM,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-continue", store),
	)

	// First call returns pending tool calls
	resp, err := a.Chat(context.Background(), "What's the weather?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 pending tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "tc-1" {
		t.Errorf("expected tool call ID tc-1, got %q", resp.ToolCalls[0].ID)
	}

	// Continue with externally-executed results
	resp, err = a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "search", Content: "It is sunny today."},
	})
	if err != nil {
		t.Fatalf("unexpected error from Continue: %v", err)
	}

	if resp.Content != "The weather is sunny." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
	if mockLLM.CallCount() != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockLLM.CallCount())
	}
}

func TestContinue_NoSession(t *testing.T) {
	mockLLM := newMockLLM()
	a := agent.New(mockLLM)

	_, err := a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Content: "result"},
	})
	if err == nil {
		t.Fatal("expected error when calling Continue without session")
	}
}

func TestContinue_EmptyResults(t *testing.T) {
	store := session.MemoryStore()
	mockLLM := newMockLLM()
	a := agent.New(mockLLM, agent.WithSession("test-empty", store))

	_, err := a.Continue(context.Background(), []message.ToolResult{})
	if err == nil {
		t.Fatal("expected error when calling Continue with empty results")
	}
}

func TestContinue_MultipleResults(t *testing.T) {
	store := session.MemoryStore()

	mockLLM := newMockLLM(
		// First Chat(): LLM requests two tool calls
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "search",
					Input: `{"q":"a"}`,
					Type:  "function",
				},
				{
					ID:    "tc-2",
					Name:  "search",
					Input: `{"q":"b"}`,
					Type:  "function",
				},
			},
		},
		// Continue: LLM produces final answer
		mockResponse{Content: "Combined results."},
	)

	a := agent.New(mockLLM,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-multi-results", store),
	)

	resp, err := a.Chat(context.Background(), "search both")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("expected 2 pending tool calls, got %d", len(resp.ToolCalls))
	}

	resp, err = a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "search", Content: "result A"},
		{ToolCallID: "tc-2", Name: "search", Content: "result B"},
	})
	if err != nil {
		t.Fatalf("unexpected error from Continue: %v", err)
	}
	if resp.Content != "Combined results." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestContinue_Stream(t *testing.T) {
	store := session.MemoryStore()

	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "search",
					Input: `{"q":"test"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "streamed result"},
	)

	a := agent.New(mockLLM,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-continue-stream", store),
	)

	// First call via stream
	var firstResp *agent.ChatResponse
	for event := range a.ChatStream(context.Background(), "stream test") {
		if event.Type == types.EventComplete && event.Response != nil {
			firstResp = event.Response
		}
	}
	if firstResp == nil {
		t.Fatal("expected response from first ChatStream")
		return
	}
	if len(firstResp.ToolCalls) != 1 {
		t.Fatalf(
			"expected 1 pending tool call, got %d",
			len(firstResp.ToolCalls),
		)
	}

	// Continue via stream
	var finalResp *agent.ChatResponse
	for event := range a.ContinueStream(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "search", Content: "search result"},
	}) {
		if event.Type == types.EventComplete && event.Response != nil {
			finalResp = event.Response
		}
	}
	if finalResp == nil {
		t.Fatal("expected response from ContinueStream")
	}
	if finalResp.Content != "streamed result" {
		t.Errorf("unexpected response: %q", finalResp.Content)
	}
}

func TestContinue_PendingToolCallsPersisted(t *testing.T) {
	store := session.MemoryStore()

	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "search",
					Input: `{"q":"test"}`,
					Type:  "function",
				},
			},
		},
	)

	a := agent.New(mockLLM,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-persist", store),
	)

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 pending tool call, got %d", len(resp.ToolCalls))
	}

	// Verify the session contains the assistant message with tool calls
	sess, err := store.Load(context.Background(), "test-persist")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	msgs, err := sess.GetMessages(context.Background(), nil)
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	// Should have: system (if any), user, assistant (with tool calls)
	var foundAssistantWithToolCalls bool
	for _, msg := range msgs {
		if msg.Role == message.Assistant {
			for _, part := range msg.Parts {
				if _, ok := part.(message.ToolCall); ok {
					foundAssistantWithToolCalls = true
					break
				}
			}
		}
	}

	if !foundAssistantWithToolCalls {
		t.Error(
			"expected session to contain assistant message with tool calls, but none found",
		)
	}
}

func TestContinueStream_BeforeAgent_ShortCircuit(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{
					Action: agent.HookModify,
					Response: &agent.ChatResponse{
						Content: "short-circuited continue",
					},
				}, nil
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
		mockResponse{Content: "unreachable"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-cs-ba", store),
		agent.WithHooks(hooks),
	)

	resp, err := a.Chat(context.Background(), "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	var finalContent string
	for event := range a.ContinueStream(
		context.Background(),
		[]message.ToolResult{
			{ToolCallID: "tc-1", Name: "echo", Content: "result"},
		},
	) {
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "short-circuited continue" {
		t.Fatalf(
			"expected 'short-circuited continue', got %q",
			finalContent,
		)
	}
}

func TestContinueStream_BeforeAgent_Error(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{}, fmt.Errorf("before-agent error")
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
		mockResponse{Content: "unreachable"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-cs-ba-err", store),
		agent.WithHooks(hooks),
	)

	resp, err := a.Chat(context.Background(), "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	var gotError bool
	for event := range a.ContinueStream(
		context.Background(),
		[]message.ToolResult{
			{ToolCallID: "tc-1", Name: "echo", Content: "result"},
		},
	) {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Fatal("expected error event from BeforeAgent in ContinueStream")
	}
}

func TestContinueStream_AfterAgent_ModifyResponse(t *testing.T) {
	store := session.MemoryStore()

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
		mockResponse{Content: "final"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-cs-aa", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	var finalContent string
	for event := range a.ContinueStream(
		context.Background(),
		[]message.ToolResult{
			{ToolCallID: "tc-1", Name: "echo", Content: "result"},
		},
	) {
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "modified: final" {
		t.Fatalf("expected 'modified: final', got %q", finalContent)
	}
}

func TestContinueStream_AfterAgent_Error(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		AfterAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{}, fmt.Errorf("after-agent error")
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
		mockResponse{Content: "final"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-cs-aa-err", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	var gotError bool
	for event := range a.ContinueStream(
		context.Background(),
		[]message.ToolResult{
			{ToolCallID: "tc-1", Name: "echo", Content: "result"},
		},
	) {
		if event.Type == types.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Fatal("expected error event from AfterAgent in ContinueStream")
	}
}

func TestContinue_BeforeAgent_ShortCircuit(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{
					Action: agent.HookModify,
					Response: &agent.ChatResponse{
						Content: "short-circuited",
					},
				}, nil
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
		mockResponse{Content: "unreachable"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-c-ba", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	resp, err := a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "echo", Content: "result"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "short-circuited" {
		t.Fatalf(
			"expected 'short-circuited', got %q",
			resp.Content,
		)
	}
}

func TestContinue_BeforeAgent_Error(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{}, fmt.Errorf("before-agent error")
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-c-ba-err", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	_, err := a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "echo", Content: "result"},
	})
	if err == nil {
		t.Fatal("expected error from BeforeAgent in Continue")
	}
}

func TestContinue_AfterAgent_ModifyResponse(t *testing.T) {
	store := session.MemoryStore()

	hooks := agent.Hooks{
		AfterAgent: func(_ context.Context, ac agent.LifecycleContext) (agent.LifecycleResult, error) {
			if ac.Response == nil {
				return agent.LifecycleResult{
					Action: agent.HookAllow,
				}, nil
			}
			modified := *ac.Response
			modified.Content = "modified: " + modified.Content
			return agent.LifecycleResult{
				Action:   agent.HookModify,
				Response: &modified,
			}, nil
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
		mockResponse{Content: "final"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-c-aa", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	resp, err := a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "echo", Content: "result"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "modified: final" {
		t.Fatalf("expected 'modified: final', got %q", resp.Content)
	}
}

func TestContinue_AfterAgent_Error(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		AfterAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{}, fmt.Errorf("after-agent error")
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
		mockResponse{Content: "final"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-c-aa-err", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	_, err := a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "echo", Content: "result"},
	})
	if err == nil {
		t.Fatal("expected error from AfterAgent in Continue")
	}
}

func TestContinue_BeforeRun_AfterRun_Fire(t *testing.T) {
	store := session.MemoryStore()

	var beforeFired, afterFired bool
	hooks := agent.Hooks{
		BeforeRun: func(_ context.Context, _ agent.RunContext) {
			beforeFired = true
		},
		AfterRun: func(_ context.Context, _ agent.RunContext) {
			afterFired = true
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
		mockResponse{Content: "final"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-c-run", store),
		agent.WithHooks(hooks),
	)

	a.Chat(context.Background(), "start")
	beforeFired = false
	afterFired = false

	a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "echo", Content: "result"},
	})

	if !beforeFired {
		t.Fatal("BeforeRun should fire in Continue")
	}
	if !afterFired {
		t.Fatal("AfterRun should fire in Continue")
	}
}

func TestContinueStream_BeforeRun_AfterRun_Fire(t *testing.T) {
	store := session.MemoryStore()

	var beforeFired, afterFired bool
	hooks := agent.Hooks{
		BeforeRun: func(_ context.Context, _ agent.RunContext) {
			beforeFired = true
		},
		AfterRun: func(_ context.Context, _ agent.RunContext) {
			afterFired = true
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
		mockResponse{Content: "final"},
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-cs-run", store),
		agent.WithHooks(hooks),
	)

	a.Chat(context.Background(), "start")
	beforeFired = false
	afterFired = false

	for event := range a.ContinueStream(
		context.Background(),
		[]message.ToolResult{
			{ToolCallID: "tc-1", Name: "echo", Content: "result"},
		},
	) {
		_ = event
	}

	if !beforeFired {
		t.Fatal("BeforeRun should fire in ContinueStream")
	}
	if !afterFired {
		t.Fatal("AfterRun should fire in ContinueStream")
	}
}

func TestContinue_BeforeAgent_Deny(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{Action: agent.HookDeny}, nil
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-c-deny", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	_, err := a.Continue(context.Background(), []message.ToolResult{
		{ToolCallID: "tc-1", Name: "echo", Content: "result"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.CallCount() != 1 {
		t.Fatal("LLM should only have been called once (Chat, not Continue)")
	}
}

func TestContinueStream_BeforeAgent_Deny(t *testing.T) {
	store := session.MemoryStore()

	callCount := 0
	hooks := agent.Hooks{
		BeforeAgent: func(_ context.Context, _ agent.LifecycleContext) (agent.LifecycleResult, error) {
			callCount++
			if callCount > 1 {
				return agent.LifecycleResult{Action: agent.HookDeny}, nil
			}
			return agent.LifecycleResult{Action: agent.HookAllow}, nil
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
	)

	a := agent.New(mock,
		agent.WithAutoExecute(false),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-cs-deny", store),
		agent.WithHooks(hooks),
	)

	resp, _ := a.Chat(context.Background(), "start")
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected pending tool calls")
	}

	var gotComplete bool
	for event := range a.ContinueStream(
		context.Background(),
		[]message.ToolResult{
			{ToolCallID: "tc-1", Name: "echo", Content: "result"},
		},
	) {
		if event.Type == types.EventComplete {
			gotComplete = true
		}
	}

	if !gotComplete {
		t.Fatal("expected complete event even on deny")
	}
}
