package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type eventCollector struct {
	mu     sync.Mutex
	events []agent.ObserverEvent
}

func (c *eventCollector) OnEvent(evt agent.ObserverEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evt)
}

func (c *eventCollector) Events() []agent.ObserverEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]agent.ObserverEvent, len(c.events))
	copy(cp, c.events)
	return cp
}

func (c *eventCollector) ByType(t agent.ObserverEventType) []agent.ObserverEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []agent.ObserverEvent
	for _, e := range c.events {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out
}

func TestObserver_TaskLifecycle_Success(t *testing.T) {
	obs := &eventCollector{}
	childLLM := newMockLLM(mockResponse{Content: "done"})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "worker", Input: `{"task":"do work","background":true}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "get_task_result", Input: `{"task_id":"task-1","wait":true}`, Type: "function"},
			},
		},
		mockResponse{Content: "all done"},
	)

	parent := agent.New(parentLLM,
		agent.WithObserver(obs),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	_, err := parent.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	launched := obs.ByType(agent.EventTaskLaunched)
	if len(launched) != 1 {
		t.Fatalf("expected 1 task_launched event, got %d", len(launched))
	}
	if launched[0].TaskID != "task-1" {
		t.Errorf("expected TaskID=task-1, got %s", launched[0].TaskID)
	}
	if launched[0].AgentName != "worker" {
		t.Errorf("expected AgentName=worker, got %s", launched[0].AgentName)
	}

	completed := obs.ByType(agent.EventTaskCompleted)
	if len(completed) != 1 {
		t.Fatalf("expected 1 task_completed event, got %d", len(completed))
	}
	if completed[0].TaskID != "task-1" {
		t.Errorf("expected TaskID=task-1, got %s", completed[0].TaskID)
	}
	if completed[0].Duration <= 0 {
		t.Error("expected Duration > 0 for task_completed")
	}
}

func TestObserver_TaskLifecycle_Failure(t *testing.T) {
	obs := &eventCollector{}
	childLLM := newMockLLM(mockResponse{Err: fmt.Errorf("child exploded")})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "worker", Input: `{"task":"fail","background":true}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "get_task_result", Input: `{"task_id":"task-1","wait":true}`, Type: "function"},
			},
		},
		mockResponse{Content: "ok"},
	)

	parent := agent.New(parentLLM,
		agent.WithObserver(obs),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Fails",
			Agent:       child,
		}),
	)

	_, err := parent.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	failed := obs.ByType(agent.EventTaskFailed)
	if len(failed) != 1 {
		t.Fatalf("expected 1 task_failed event, got %d", len(failed))
	}
	if failed[0].Error == "" {
		t.Error("expected Error to be set on task_failed event")
	}
	if failed[0].Duration <= 0 {
		t.Error("expected Duration > 0 for task_failed")
	}
}

func TestObserver_TaskLifecycle_Cancellation(t *testing.T) {
	obs := &eventCollector{}
	childLLM := &blockingMockLLM{
		delay:    10 * time.Second,
		fallback: newMockLLM(mockResponse{Content: "never"}),
	}
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "worker", Input: `{"task":"block","background":true}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "stop_task", Input: `{"task_id":"task-1"}`, Type: "function"},
			},
		},
		mockResponse{Content: "stopped"},
	)

	parent := agent.New(parentLLM,
		agent.WithObserver(obs),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Blocks",
			Agent:       child,
		}),
	)

	_, err := parent.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	cancelled := obs.ByType(agent.EventTaskCancelled)
	if len(cancelled) != 1 {
		t.Fatalf("expected 1 task_cancelled event, got %d", len(cancelled))
	}
	if cancelled[0].TaskID != "task-1" {
		t.Errorf("expected TaskID=task-1, got %s", cancelled[0].TaskID)
	}
}

func TestObserver_TurnLifecycle(t *testing.T) {
	obs := &eventCollector{}

	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"hello"}`, Type: "function"},
			},
			Usage: llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
		},
		mockResponse{
			Content: "final",
			Usage:   llm.TokenUsage{InputTokens: 20, OutputTokens: 10},
		},
	)

	a := agent.New(mockLLM,
		agent.WithObserver(obs),
		agent.WithTools(&echoTool{}),
	)

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	started := obs.ByType(agent.EventTurnStarted)
	completed := obs.ByType(agent.EventTurnCompleted)

	if len(started) != 2 {
		t.Fatalf("expected 2 turn_started events, got %d", len(started))
	}
	if len(completed) != 2 {
		t.Fatalf("expected 2 turn_completed events, got %d", len(completed))
	}

	if started[0].TurnIndex != 0 {
		t.Errorf("expected first turn index 0, got %d", started[0].TurnIndex)
	}
	if started[1].TurnIndex != 1 {
		t.Errorf("expected second turn index 1, got %d", started[1].TurnIndex)
	}

	if completed[0].TurnIndex != 0 {
		t.Errorf("expected first completed turn index 0, got %d", completed[0].TurnIndex)
	}
	if completed[0].ToolCount != 1 {
		t.Errorf("expected ToolCount=1 on first turn, got %d", completed[0].ToolCount)
	}
	if completed[0].Duration <= 0 {
		t.Error("expected Duration > 0 for turn_completed")
	}
}

func TestObserver_ToolLifecycle_Success(t *testing.T) {
	obs := &eventCollector{}

	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"hello"}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mockLLM,
		agent.WithObserver(obs),
		agent.WithTools(&echoTool{}),
	)

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	started := obs.ByType(agent.EventToolStarted)
	succeeded := obs.ByType(agent.EventToolSucceeded)

	if len(started) != 1 {
		t.Fatalf("expected 1 tool_started event, got %d", len(started))
	}
	if len(succeeded) != 1 {
		t.Fatalf("expected 1 tool_succeeded event, got %d", len(succeeded))
	}

	if started[0].ToolName != "echo" {
		t.Errorf("expected ToolName=echo, got %s", started[0].ToolName)
	}
	if started[0].ToolCallID != "tc-1" {
		t.Errorf("expected ToolCallID=tc-1, got %s", started[0].ToolCallID)
	}
	if succeeded[0].Duration <= 0 {
		t.Error("expected Duration > 0 for tool_succeeded")
	}
}

type failingTool struct{}

func (t *failingTool) Info() tool.ToolInfo {
	return tool.NewToolInfo("fail_tool", "Always fails", struct{}{})
}

func (t *failingTool) Run(_ context.Context, _ tool.ToolCall) (tool.ToolResponse, error) {
	return tool.NewTextErrorResponse("tool broke"), nil
}

func TestObserver_ToolLifecycle_Error(t *testing.T) {
	obs := &eventCollector{}

	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "fail_tool", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "handled"},
	)

	a := agent.New(mockLLM,
		agent.WithObserver(obs),
		agent.WithTools(&failingTool{}),
	)

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errored := obs.ByType(agent.EventToolErrored)
	if len(errored) != 1 {
		t.Fatalf("expected 1 tool_errored event, got %d", len(errored))
	}
	if errored[0].Error == "" {
		t.Error("expected Error to be set on tool_errored event")
	}
	if errored[0].ToolName != "fail_tool" {
		t.Errorf("expected ToolName=fail_tool, got %s", errored[0].ToolName)
	}
}

func TestObserver_MultiTurnEventOrdering(t *testing.T) {
	obs := &eventCollector{}

	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"a"}`, Type: "function"},
			},
		},
		mockResponse{Content: "final"},
	)

	a := agent.New(mockLLM,
		agent.WithObserver(obs),
		agent.WithTools(&echoTool{}),
	)

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := obs.Events()
	expected := []agent.ObserverEventType{
		agent.EventTurnStarted,
		agent.EventTurnCompleted,
		agent.EventToolStarted,
		agent.EventToolSucceeded,
		agent.EventTurnStarted,
		agent.EventTurnCompleted,
	}

	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(events), eventTypes(events))
	}

	for i, exp := range expected {
		if events[i].Type != exp {
			t.Errorf("event[%d]: expected %s, got %s", i, exp, events[i].Type)
		}
	}
}

func TestObserver_NilObserver_NoPanic(t *testing.T) {
	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"a"}`, Type: "function"},
			},
		},
		mockResponse{Content: "ok"},
	)

	a := agent.New(mockLLM, agent.WithTools(&echoTool{}))

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObserver_MultiObserver(t *testing.T) {
	obs1 := &eventCollector{}
	obs2 := &eventCollector{}
	multi := agent.MultiObserver{obs1, obs2}

	mockLLM := newMockLLM(mockResponse{Content: "hi"})
	a := agent.New(mockLLM, agent.WithObserver(multi))

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(obs1.Events()) == 0 {
		t.Error("expected obs1 to receive events")
	}
	if len(obs1.Events()) != len(obs2.Events()) {
		t.Errorf("expected both observers to receive same number of events: %d vs %d",
			len(obs1.Events()), len(obs2.Events()))
	}
}

func TestObserver_MultiAgentCorrelation(t *testing.T) {
	obs := &eventCollector{}

	child1LLM := newMockLLM(mockResponse{Content: "r1"})
	child2LLM := newMockLLM(mockResponse{Content: "r2"})
	child3LLM := newMockLLM(mockResponse{Content: "r3"})

	child1 := agent.New(child1LLM)
	child2 := agent.New(child2LLM)
	child3 := agent.New(child3LLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "w1", Input: `{"task":"a","background":true}`, Type: "function"},
				{ID: "tc-2", Name: "w2", Input: `{"task":"b","background":true}`, Type: "function"},
				{ID: "tc-3", Name: "w3", Input: `{"task":"c","background":true}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-4", Name: "get_task_result", Input: `{"task_id":"task-1","wait":true}`, Type: "function"},
				{ID: "tc-5", Name: "get_task_result", Input: `{"task_id":"task-2","wait":true}`, Type: "function"},
				{ID: "tc-6", Name: "get_task_result", Input: `{"task_id":"task-3","wait":true}`, Type: "function"},
			},
		},
		mockResponse{Content: "all done"},
	)

	parent := agent.New(parentLLM,
		agent.WithObserver(obs),
		agent.WithSubAgents(
			agent.SubAgentConfig{Name: "w1", Description: "Worker 1", Agent: child1},
			agent.SubAgentConfig{Name: "w2", Description: "Worker 2", Agent: child2},
			agent.SubAgentConfig{Name: "w3", Description: "Worker 3", Agent: child3},
		),
	)

	_, err := parent.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	launched := obs.ByType(agent.EventTaskLaunched)
	if len(launched) != 3 {
		t.Fatalf("expected 3 task_launched events, got %d", len(launched))
	}

	completed := obs.ByType(agent.EventTaskCompleted)
	if len(completed) != 3 {
		t.Fatalf("expected 3 task_completed events, got %d", len(completed))
	}

	turnsByTask := make(map[string][]agent.ObserverEvent)
	for _, evt := range obs.ByType(agent.EventTurnStarted) {
		if evt.TaskID != "" {
			turnsByTask[evt.TaskID] = append(turnsByTask[evt.TaskID], evt)
		}
	}

	for _, taskID := range []string{"task-1", "task-2", "task-3"} {
		if len(turnsByTask[taskID]) == 0 {
			t.Errorf("expected turn events for %s, got none", taskID)
		}
	}
}

func TestObserver_AutoPropagation(t *testing.T) {
	obs := &eventCollector{}
	childLLM := newMockLLM(mockResponse{Content: "child done"})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "worker", Input: `{"task":"sync work"}`, Type: "function"},
			},
		},
		mockResponse{Content: "parent done"},
	)

	parent := agent.New(parentLLM,
		agent.WithObserver(obs),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Sync worker",
			Agent:       child,
		}),
	)

	_, err := parent.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	allTurns := obs.ByType(agent.EventTurnStarted)
	if len(allTurns) < 3 {
		t.Errorf("expected at least 3 turn_started events (2 parent + 1 child), got %d", len(allTurns))
	}
}

func TestObserver_StreamTurnLifecycle(t *testing.T) {
	obs := &eventCollector{}

	mockLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"hello"}`, Type: "function"},
			},
			Usage: llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
		},
		mockResponse{
			Content: "final",
			Usage:   llm.TokenUsage{InputTokens: 20, OutputTokens: 10},
		},
	)

	a := agent.New(mockLLM,
		agent.WithObserver(obs),
		agent.WithTools(&echoTool{}),
	)

	for event := range a.ChatStream(context.Background(), "hi") {
		if event.Type == types.EventError {
			t.Fatalf("unexpected stream error: %v", event.Error)
		}
	}

	started := obs.ByType(agent.EventTurnStarted)
	completed := obs.ByType(agent.EventTurnCompleted)

	if len(started) != 2 {
		t.Fatalf("expected 2 turn_started events in stream, got %d", len(started))
	}
	if len(completed) != 2 {
		t.Fatalf("expected 2 turn_completed events in stream, got %d", len(completed))
	}
}

func eventTypes(events []agent.ObserverEvent) []agent.ObserverEventType {
	out := make([]agent.ObserverEventType, len(events))
	for i, e := range events {
		out[i] = e.Type
	}
	return out
}
