package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type hookEventCollector struct {
	mu     sync.Mutex
	events []agent.HookEvent
}

func (c *hookEventCollector) collect(evt agent.HookEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evt)
}

func (c *hookEventCollector) all() []agent.HookEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]agent.HookEvent, len(c.events))
	copy(cp, c.events)
	return cp
}

func (c *hookEventCollector) ofType(t agent.HookEventType) []agent.HookEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []agent.HookEvent
	for _, evt := range c.events {
		if evt.Type == t {
			result = append(result, evt)
		}
	}
	return result
}

func TestPreToolUse_Deny(t *testing.T) {
	toolExecuted := false
	deniedTool := &simpleTool{
		name: "forbidden",
		run: func(_ context.Context, _ tool.ToolCall) (tool.ToolResponse, error) {
			toolExecuted = true
			return tool.NewTextResponse("should not happen"), nil
		},
	}

	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
			if tc.ToolName == "forbidden" {
				return agent.PreToolUseResult{
					Action:     agent.HookDeny,
					DenyReason: "not allowed",
				}, nil
			}
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "forbidden",
					Input: `{"text":"hi"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "ok denied"},
	)

	a := agent.New(mock,
		agent.WithTools(deniedTool),
		agent.WithHooks(hooks),
	)

	resp, err := a.Chat(context.Background(), "try forbidden")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if toolExecuted {
		t.Fatal("tool should not have been executed when denied by hook")
	}
	if resp.Content != "ok denied" {
		t.Fatalf("expected 'ok denied', got %q", resp.Content)
	}
}

func TestPreToolUse_Modify(t *testing.T) {
	var capturedInput string
	modTool := &simpleTool{
		name: "echo",
		run: func(_ context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
			capturedInput = params.Input
			return tool.NewTextResponse("echoed"), nil
		},
	}

	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
			return agent.PreToolUseResult{
				Action: agent.HookModify,
				Input:  `{"text":"modified"}`,
			}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"original"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(modTool),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedInput != `{"text":"modified"}` {
		t.Fatalf("expected modified input, got %q", capturedInput)
	}
}

func TestPreToolUse_Allow(t *testing.T) {
	toolExecuted := false
	simpleTl := &simpleTool{
		name: "ping",
		run: func(_ context.Context, _ tool.ToolCall) (tool.ToolResponse, error) {
			toolExecuted = true
			return tool.NewTextResponse("pong"), nil
		},
	}

	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "ping", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(simpleTl),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "ping it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !toolExecuted {
		t.Fatal("tool should have been executed when allowed")
	}
}

func TestPostToolUse_Modify(t *testing.T) {
	var capturedToolResult string
	echoTl := &echoTool{}

	hooks := agent.Hooks{
		PostToolUse: func(_ context.Context, tc agent.PostToolUseContext) (agent.PostToolUseResult, error) {
			return agent.PostToolUseResult{
				Action: agent.HookModify,
				Output: "intercepted output",
			}, nil
		},
	}

	parentBase := newMockLLM(
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
		mockResponse{Content: "final"},
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
		agent.WithTools(echoTl),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedToolResult != "intercepted output" {
		t.Fatalf("expected 'intercepted output', got %q", capturedToolResult)
	}
}

func TestPreModelCall_Modify(t *testing.T) {
	var capturedMsgCount int
	hooks := agent.Hooks{
		PreModelCall: func(_ context.Context, mc agent.ModelCallContext) (agent.ModelCallResult, error) {
			extra := message.Message{
				Role:      message.System,
				CreatedAt: time.Now().UnixNano(),
			}
			extra.AppendContent("injected by hook")
			msgs := append(mc.Messages, extra)
			capturedMsgCount = len(msgs)
			return agent.ModelCallResult{
				Action:   agent.HookModify,
				Messages: msgs,
				Tools:    mc.Tools,
			}, nil
		},
	}

	mock := newMockLLM(mockResponse{Content: "done"})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMsgCount < 2 {
		t.Fatalf(
			"expected at least 2 messages (original + injected), got %d",
			capturedMsgCount,
		)
	}
}

func TestPostModelCall_Modify(t *testing.T) {
	hooks := agent.Hooks{
		PostModelCall: func(_ context.Context, mc agent.ModelResponseContext) (agent.ModelResponseResult, error) {
			modified := *mc.Response
			modified.Content = "overridden by hook"
			return agent.ModelResponseResult{
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
	if resp.Content != "overridden by hook" {
		t.Fatalf("expected 'overridden by hook', got %q", resp.Content)
	}
}

func TestPostModelCall_OnError(t *testing.T) {
	var hookFired bool
	var hookError error
	hooks := agent.Hooks{
		PostModelCall: func(_ context.Context, mc agent.ModelResponseContext) (agent.ModelResponseResult, error) {
			hookFired = true
			hookError = mc.Error
			return agent.ModelResponseResult{Action: agent.HookAllow}, nil
		},
	}

	mock := newMockLLM(mockResponse{Err: fmt.Errorf("llm exploded")})
	a := agent.New(mock, agent.WithHooks(hooks))

	_, err := a.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from LLM")
	}
	if !hookFired {
		t.Fatal("PostModelCall hook should fire even on LLM error")
	}
	if hookError == nil || hookError.Error() != "llm exploded" {
		t.Fatalf("expected hook to receive LLM error, got %v", hookError)
	}
}

func TestHookChaining_DenyWins(t *testing.T) {
	toolExecuted := false
	simpleTl := &simpleTool{
		name: "target",
		run: func(_ context.Context, _ tool.ToolCall) (tool.ToolResponse, error) {
			toolExecuted = true
			return tool.NewTextResponse("ran"), nil
		},
	}

	hook1 := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}
	hook2 := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			return agent.PreToolUseResult{
				Action:     agent.HookDeny,
				DenyReason: "blocked",
			}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "target", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(simpleTl),
		agent.WithHooks(hook1, hook2),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if toolExecuted {
		t.Fatal("tool should not execute when second hook denies")
	}
}

func TestHookChaining_LastModifyWins(t *testing.T) {
	var capturedInput string
	simpleTl := &simpleTool{
		name: "target",
		run: func(_ context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
			capturedInput = params.Input
			return tool.NewTextResponse("ok"), nil
		},
	}

	hook1 := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			return agent.PreToolUseResult{
				Action: agent.HookModify,
				Input:  `{"v":"first"}`,
			}, nil
		},
	}
	hook2 := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			return agent.PreToolUseResult{
				Action: agent.HookModify,
				Input:  `{"v":"second"}`,
			}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "target",
					Input: `{"v":"original"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(simpleTl),
		agent.WithHooks(hook1, hook2),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedInput != `{"v":"second"}` {
		t.Fatalf("expected last modify to win, got %q", capturedInput)
	}
}

func TestOnSubagentStart_OnSubagentStop(t *testing.T) {
	collector := &hookEventCollector{}

	childLLM := newMockLLM(mockResponse{Content: "child done"})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"do it","background":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "get_task_result",
					Input: `{"task_id":"task-1","wait":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "all done"},
	)

	a := agent.New(parentLLM,
		agent.WithHooks(agent.NewObservingHooks(collector.collect)),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := a.Chat(context.Background(), "launch worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "all done" {
		t.Fatalf("expected 'all done', got %q", resp.Content)
	}

	starts := collector.ofType(agent.HookEventSubagentStart)
	stops := collector.ofType(agent.HookEventSubagentStop)

	if len(starts) != 1 {
		t.Fatalf("expected 1 subagent_start event, got %d", len(starts))
	}
	if len(stops) != 1 {
		t.Fatalf("expected 1 subagent_stop event, got %d", len(stops))
	}
	if starts[0].TaskID == "" {
		t.Fatal("subagent_start should have TaskID")
	}
	if starts[0].AgentName != "worker" {
		t.Fatalf("expected AgentName 'worker', got %q", starts[0].AgentName)
	}
	if stops[0].Error != "" {
		t.Fatalf("expected no error on stop, got %q", stops[0].Error)
	}
}

func TestOnSubagentStop_WithError(t *testing.T) {
	collector := &hookEventCollector{}

	childLLM := newMockLLM(mockResponse{Err: fmt.Errorf("child exploded")})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"fail","background":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "get_task_result",
					Input: `{"task_id":"task-1","wait":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "handled"},
	)

	a := agent.New(parentLLM,
		agent.WithHooks(agent.NewObservingHooks(collector.collect)),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Fails",
			Agent:       child,
		}),
	)

	_, err := a.Chat(context.Background(), "launch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stops := collector.ofType(agent.HookEventSubagentStop)
	if len(stops) != 1 {
		t.Fatalf("expected 1 subagent_stop event, got %d", len(stops))
	}
	if stops[0].Error == "" {
		t.Fatal("expected error on subagent_stop for failed child")
	}
}

func TestNilHooks_NoPanic(t *testing.T) {
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

	a := agent.New(mock, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Fatalf("expected 'done', got %q", resp.Content)
	}
}

func TestHooksPropagateToSubAgents(t *testing.T) {
	var toolHookFired bool
	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
			if tc.ToolName == "echo" {
				toolHookFired = true
			}
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}

	childLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-child",
					Name:  "echo",
					Input: `{"text":"from child"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "child done"},
	)
	child := agent.New(childLLM, agent.WithTools(&echoTool{}))

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"run echo"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "parent done"},
	)

	a := agent.New(parentLLM,
		agent.WithHooks(hooks),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Worker",
			Agent:       child,
		}),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !toolHookFired {
		t.Fatal(
			"parent hook should have propagated to sub-agent and fired for echo tool",
		)
	}
}

func TestNewObservingHooks(t *testing.T) {
	collector := &hookEventCollector{}

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
		agent.WithHooks(agent.NewObservingHooks(collector.collect)),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := collector.all()
	if len(events) == 0 {
		t.Fatal("expected at least some events from observing hooks")
	}

	typesSeen := make(map[agent.HookEventType]bool)
	for _, evt := range events {
		typesSeen[evt.Type] = true
	}

	expected := []agent.HookEventType{
		agent.HookEventPreModelCall,
		agent.HookEventPostModelCall,
		agent.HookEventPreToolUse,
		agent.HookEventPostToolUse,
	}
	for _, et := range expected {
		if !typesSeen[et] {
			t.Errorf("expected event type %q to be observed", et)
		}
	}

	toolEvents := collector.ofType(agent.HookEventPreToolUse)
	if len(toolEvents) > 0 && toolEvents[0].ToolName != "echo" {
		t.Errorf("expected ToolName 'echo', got %q", toolEvents[0].ToolName)
	}

	postModel := collector.ofType(agent.HookEventPostModelCall)
	if len(postModel) > 0 && postModel[0].Duration == 0 {
		t.Error("expected Duration > 0 on post_model_call events")
	}
}

func TestPreToolUse_ErrorTreatedAsDeny(t *testing.T) {
	toolExecuted := false
	simpleTl := &simpleTool{
		name: "target",
		run: func(_ context.Context, _ tool.ToolCall) (tool.ToolResponse, error) {
			toolExecuted = true
			return tool.NewTextResponse("ran"), nil
		},
	}

	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			return agent.PreToolUseResult{}, fmt.Errorf("hook crashed")
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "target", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(simpleTl),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if toolExecuted {
		t.Fatal("tool should not execute when hook returns error")
	}
}

func TestHooksWithStreaming(t *testing.T) {
	collector := &hookEventCollector{}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "echo",
					Input: `{"text":"stream"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "streamed"},
	)

	a := agent.New(mock,
		agent.WithTools(&echoTool{}),
		agent.WithHooks(agent.NewObservingHooks(collector.collect)),
	)

	var finalContent string
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "streamed" {
		t.Fatalf("expected 'streamed', got %q", finalContent)
	}

	events := collector.all()
	if len(events) == 0 {
		t.Fatal("expected hook events from streaming path")
	}

	typesSeen := make(map[agent.HookEventType]bool)
	for _, evt := range events {
		typesSeen[evt.Type] = true
	}

	if !typesSeen[agent.HookEventPreModelCall] {
		t.Error("expected pre_model_call event in streaming path")
	}
	if !typesSeen[agent.HookEventPreToolUse] {
		t.Error("expected pre_tool_use event in streaming path")
	}
}

func TestHooksWithParallelTools(t *testing.T) {
	var mu sync.Mutex
	var toolNames []string

	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
			mu.Lock()
			toolNames = append(toolNames, tc.ToolName)
			mu.Unlock()
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}

	tool1 := &simpleTool{
		name: "tool_a",
		run: func(_ context.Context, _ tool.ToolCall) (tool.ToolResponse, error) {
			time.Sleep(10 * time.Millisecond)
			return tool.NewTextResponse("a"), nil
		},
	}
	tool2 := &simpleTool{
		name: "tool_b",
		run: func(_ context.Context, _ tool.ToolCall) (tool.ToolResponse, error) {
			time.Sleep(10 * time.Millisecond)
			return tool.NewTextResponse("b"), nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "tool_a", Input: `{}`, Type: "function"},
				{ID: "tc-2", Name: "tool_b", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock,
		agent.WithTools(tool1, tool2),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(toolNames) != 2 {
		t.Fatalf("expected 2 PreToolUse calls, got %d", len(toolNames))
	}

	seen := make(map[string]bool)
	for _, n := range toolNames {
		seen[n] = true
	}
	if !seen["tool_a"] || !seen["tool_b"] {
		t.Fatalf("expected both tool_a and tool_b, got %v", toolNames)
	}
}

func TestBranch_OnObserverEvents(t *testing.T) {
	collector := &hookEventCollector{}

	childLLM := newMockLLM(mockResponse{Content: "child done"})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"do it","background":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "get_task_result",
					Input: `{"task_id":"task-1","wait":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "all done"},
	)

	a := agent.New(parentLLM,
		agent.WithHooks(agent.NewObservingHooks(collector.collect)),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	_, err := a.Chat(context.Background(), "launch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Child model call events should have a TaskID set
	modelCalls := collector.ofType(agent.HookEventPreModelCall)
	var childModelCalls []agent.HookEvent
	for _, evt := range modelCalls {
		if evt.TaskID != "" {
			childModelCalls = append(childModelCalls, evt)
		}
	}
	if len(childModelCalls) == 0 {
		t.Fatal(
			"expected child model call events to have TaskID for correlation",
		)
	}
}

func TestMultipleHookChains(t *testing.T) {
	var order []string
	var mu sync.Mutex

	hook1 := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			mu.Lock()
			order = append(order, "hook1")
			mu.Unlock()
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}
	hook2 := agent.Hooks{
		PreToolUse: func(_ context.Context, _ agent.ToolUseContext) (agent.PreToolUseResult, error) {
			mu.Lock()
			order = append(order, "hook2")
			mu.Unlock()
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
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
		agent.WithHooks(hook1, hook2),
	)

	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "hook1" || order[1] != "hook2" {
		t.Fatalf("expected hooks to run in order [hook1, hook2], got %v", order)
	}
}

// simpleTool is a test helper for creating tools with custom behavior.
type simpleTool struct {
	name string
	run  func(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error)
}

func (t *simpleTool) Info() tool.ToolInfo {
	return tool.NewToolInfo(t.name, "A test tool", struct {
		Text string `json:"text" desc:"Input text" required:"false"`
	}{})
}

func (t *simpleTool) Run(
	ctx context.Context,
	params tool.ToolCall,
) (tool.ToolResponse, error) {
	return t.run(ctx, params)
}
