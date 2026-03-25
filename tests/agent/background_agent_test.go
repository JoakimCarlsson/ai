package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

func TestBackground_Launch(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Content: "child result"})
	child := agent.New(childLLM)

	var capturedToolResult string
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"do work","background":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "background launched"},
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

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "launch in background")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "background launched" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if !strings.Contains(capturedToolResult, `"status":"launched"`) {
		t.Errorf(
			"expected status:launched in tool result, got: %s",
			capturedToolResult,
		)
	}
	if !strings.Contains(capturedToolResult, `"task_id"`) {
		t.Errorf("expected task_id in tool result, got: %s", capturedToolResult)
	}
}

func TestBackground_GetResultWait(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Content: "research findings"})
	child := agent.New(childLLM)

	var allToolResults []string
	var mu sync.Mutex
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "researcher",
					Input: `{"task":"research topic","background":true}`,
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
		mockResponse{Content: "final answer"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							mu.Lock()
							allToolResults = append(allToolResults, tr.Content)
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "researcher",
			Description: "Researches topics",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "research in background")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "final answer" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(allToolResults) < 2 {
		t.Fatalf(
			"expected at least 2 tool results, got %d",
			len(allToolResults),
		)
	}

	var foundResult bool
	for _, tr := range allToolResults {
		if strings.Contains(tr, "research findings") {
			foundResult = true
		}
	}
	if !foundResult {
		t.Errorf(
			"expected child result 'research findings' in get_task_result output, got: %v",
			allToolResults,
		)
	}
}

func TestBackground_GetResultNoWait(t *testing.T) {
	// Child that takes a while (we simulate by having it available but using no-wait polling)
	childLLM := newMockLLM(mockResponse{Content: "slow result"})
	child := agent.New(childLLM)

	var toolResults []string
	var mu sync.Mutex
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"slow work","background":true}`,
					Type:  "function",
				},
			},
		},
		// Poll without wait — might get "running" or "completed" depending on timing
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "get_task_result",
					Input: `{"task_id":"task-1","wait":false}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done polling"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							mu.Lock()
							toolResults = append(toolResults, tr.Content)
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test no-wait poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "done polling" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()

	// The poll result should contain a valid status (either "running" or "completed")
	if len(toolResults) < 2 {
		t.Fatalf("expected at least 2 tool results, got %d", len(toolResults))
	}
	pollResult := toolResults[1]
	if !strings.Contains(pollResult, `"status"`) {
		t.Errorf("expected status field in poll result, got: %s", pollResult)
	}
}

func TestBackground_StopTask(t *testing.T) {
	// Child that blocks until context is cancelled
	blockingLLM := &blockingMockLLM{
		delay:    5 * time.Second,
		fallback: newMockLLM(mockResponse{Content: "should not complete"}),
	}
	child := agent.New(blockingLLM)

	var toolResults []string
	var mu sync.Mutex
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"slow","background":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "stop_task",
					Input: `{"task_id":"task-1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-3",
					Name:  "get_task_result",
					Input: `{"task_id":"task-1","wait":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "task stopped"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							mu.Lock()
							toolResults = append(toolResults, tr.Content)
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test stop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "task stopped" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()

	// The get_task_result after stop should show cancelled status
	var foundCancelled bool
	for _, tr := range toolResults {
		if strings.Contains(tr, `"cancelled"`) {
			foundCancelled = true
		}
	}
	if !foundCancelled {
		t.Errorf("expected cancelled status in results, got: %v", toolResults)
	}
}

func TestBackground_MultipleTasks(t *testing.T) {
	childALLM := newMockLLM(mockResponse{Content: "result A"})
	childA := agent.New(childALLM)

	childBLLM := newMockLLM(mockResponse{Content: "result B"})
	childB := agent.New(childBLLM)

	var toolResults []string
	var mu sync.Mutex
	parentBase := newMockLLM(
		// Launch both in one turn
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "agent_a",
					Input: `{"task":"task A","background":true}`,
					Type:  "function",
				},
				{
					ID:    "tc-2",
					Name:  "agent_b",
					Input: `{"task":"task B","background":true}`,
					Type:  "function",
				},
			},
		},
		// Collect both results
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-3",
					Name:  "get_task_result",
					Input: `{"task_id":"task-1","wait":true}`,
					Type:  "function",
				},
				{
					ID:    "tc-4",
					Name:  "get_task_result",
					Input: `{"task_id":"task-2","wait":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "both results collected"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							mu.Lock()
							toolResults = append(toolResults, tr.Content)
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(
			agent.SubAgentConfig{
				Name:        "agent_a",
				Description: "Agent A",
				Agent:       childA,
			},
			agent.SubAgentConfig{
				Name:        "agent_b",
				Description: "Agent B",
				Agent:       childB,
			},
		),
	)

	resp, err := parent.Chat(context.Background(), "launch both")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "both results collected" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()

	var foundA, foundB bool
	for _, tr := range toolResults {
		if strings.Contains(tr, "result A") {
			foundA = true
		}
		if strings.Contains(tr, "result B") {
			foundB = true
		}
	}
	if !foundA {
		t.Error("expected 'result A' in collected results")
	}
	if !foundB {
		t.Error("expected 'result B' in collected results")
	}
}

func TestBackground_SyncUnchanged(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Content: "sync child result"})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"do work"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "sync done"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "sync test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "sync done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if childLLM.CallCount() != 1 {
		t.Errorf(
			"expected child to be called once, got %d",
			childLLM.CallCount(),
		)
	}
}

func TestBackground_TaskNotFound(t *testing.T) {
	childLLM := newMockLLM()
	child := agent.New(childLLM)

	var capturedToolResult string
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "get_task_result",
					Input: `{"task_id":"nonexistent","wait":false}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "handled not found"},
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

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "check nonexistent task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "handled not found" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if !strings.Contains(capturedToolResult, "not found") {
		t.Errorf(
			"expected 'not found' error in tool result, got: %s",
			capturedToolResult,
		)
	}
}

func TestBackground_CleanupOnChatReturn(t *testing.T) {
	// Child that takes a while — we track if it gets cancelled
	var childCancelled bool
	var childMu sync.Mutex
	blockingLLM := &blockingMockLLM{
		delay:    5 * time.Second,
		fallback: newMockLLM(mockResponse{Content: "should not finish"}),
		onCancel: func() {
			childMu.Lock()
			childCancelled = true
			childMu.Unlock()
		},
	}
	child := agent.New(blockingLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"long work","background":true}`,
					Type:  "function",
				},
			},
		},
		// Parent finishes without collecting the result
		mockResponse{Content: "done without collecting"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "launch and leave")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "done without collecting" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	// After Chat() returns, the background task should have been cancelled
	childMu.Lock()
	defer childMu.Unlock()
	if !childCancelled {
		t.Error("expected background task to be cancelled on Chat() return")
	}
}

func TestBackground_Stream(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Content: "streamed child result"})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"stream work","background":true}`,
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
		mockResponse{Content: "stream final"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	var finalContent string
	var toolResultContents []string
	for event := range parent.ChatStream(context.Background(), "stream background test") {
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
		if event.ToolResult != nil {
			toolResultContents = append(
				toolResultContents,
				event.ToolResult.Output,
			)
		}
	}

	if finalContent != "stream final" {
		t.Errorf("unexpected final content: %q", finalContent)
	}

	var foundChildResult bool
	for _, tr := range toolResultContents {
		if strings.Contains(tr, "streamed child result") {
			foundChildResult = true
		}
	}
	if !foundChildResult {
		t.Error("expected child result in stream tool results")
	}
}

func TestBackground_TaskToolsAutoRegistered(t *testing.T) {
	childLLM := newMockLLM()
	child := agent.New(childLLM)

	// The parent LLM captures the tools it receives
	var receivedToolNames []string
	var mu sync.Mutex
	parentBase := newMockLLM(mockResponse{Content: "done"})
	parentLLM := &toolCapturingLLM{
		base: parentBase,
		onTools: func(tools []string) {
			mu.Lock()
			receivedToolNames = tools
			mu.Unlock()
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	_, err := parent.Chat(context.Background(), "check tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	var hasGetResult, hasStopTask, hasWorker bool
	for _, name := range receivedToolNames {
		switch name {
		case "get_task_result":
			hasGetResult = true
		case "stop_task":
			hasStopTask = true
		case "worker":
			hasWorker = true
		}
	}

	if !hasGetResult {
		t.Error("expected get_task_result tool to be auto-registered")
	}
	if !hasStopTask {
		t.Error("expected stop_task tool to be auto-registered")
	}
	if !hasWorker {
		t.Error("expected worker sub-agent tool")
	}
}

func TestBackground_GetResultTimeout(t *testing.T) {
	// Child that blocks for a long time
	blockingLLM := &blockingMockLLM{
		delay:    5 * time.Second,
		fallback: newMockLLM(mockResponse{Content: "slow result"}),
	}
	child := agent.New(blockingLLM)

	var toolResults []string
	var mu sync.Mutex
	parentBase := newMockLLM(
		// Launch background task
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"slow work","background":true}`,
					Type:  "function",
				},
			},
		},
		// Wait with a short timeout — should return "running" since task won't finish in 100ms
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "get_task_result",
					Input: `{"task_id":"task-1","wait":true,"timeout":100}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "timed out polling"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							mu.Lock()
							toolResults = append(toolResults, tr.Content)
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	start := time.Now()
	resp, err := parent.Chat(context.Background(), "test timeout")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "timed out polling" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	// Should return quickly (within ~1s), not block for 5s
	if elapsed > 2*time.Second {
		t.Errorf("expected quick return with timeout, but took %v", elapsed)
	}

	mu.Lock()
	defer mu.Unlock()

	// The get_task_result should have returned with status "running" (task didn't finish).
	// The callback captures tool results from ALL messages on each LLM call, so the
	// get_task_result response is the last captured result.
	var foundRunning bool
	for _, tr := range toolResults {
		if strings.Contains(tr, `"status":"running"`) {
			foundRunning = true
		}
	}
	if !foundRunning {
		t.Errorf(
			"expected status:running after timeout in at least one tool result, got: %v",
			toolResults,
		)
	}
}

func TestBackground_ListTasks(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Content: "child done"})
	child := agent.New(childLLM)

	var listResult string
	var mu sync.Mutex
	parentBase := newMockLLM(
		// Launch a background task
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"some work","background":true}`,
					Type:  "function",
				},
			},
		},
		// List all tasks
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "list_tasks", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "listed tasks"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							mu.Lock()
							// Keep overwriting — we want the last tool result (list_tasks)
							listResult = tr.Content
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "list tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "listed tasks" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	mu.Lock()
	defer mu.Unlock()

	// list_tasks should return a JSON array with the task
	if !strings.Contains(listResult, `"task_id"`) {
		t.Errorf("expected task_id in list result, got: %s", listResult)
	}
	if !strings.Contains(listResult, `"agent_name"`) {
		t.Errorf("expected agent_name in list result, got: %s", listResult)
	}
	if !strings.Contains(listResult, `"worker"`) {
		t.Errorf(
			"expected agent name 'worker' in list result, got: %s",
			listResult,
		)
	}

	// Verify it's a valid JSON array
	var tasks []json.RawMessage
	if err := json.Unmarshal([]byte(listResult), &tasks); err != nil {
		t.Errorf(
			"expected valid JSON array from list_tasks, got error: %v, content: %s",
			err,
			listResult,
		)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task in list, got %d", len(tasks))
	}
}

func TestBackground_StopTaskJSON(t *testing.T) {
	blockingLLM := &blockingMockLLM{
		delay:    5 * time.Second,
		fallback: newMockLLM(mockResponse{Content: "blocked"}),
	}
	child := agent.New(blockingLLM)

	var stopResult string
	var mu sync.Mutex
	parentBase := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"slow","background":true}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "stop_task",
					Input: `{"task_id":"task-1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "stopped"},
	)
	parentLLM := &toolResultCapturingLLM{
		base: parentBase,
		onCall: func(msgs []message.Message) {
			for _, msg := range msgs {
				if msg.Role == "tool" {
					for _, part := range msg.Parts {
						if tr, ok := part.(message.ToolResult); ok {
							mu.Lock()
							stopResult = tr.Content
							mu.Unlock()
						}
					}
				}
			}
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	_, err := parent.Chat(context.Background(), "stop task test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify stop_task returns structured JSON
	var result map[string]string
	if err := json.Unmarshal([]byte(stopResult), &result); err != nil {
		t.Fatalf(
			"expected valid JSON from stop_task, got error: %v, content: %s",
			err,
			stopResult,
		)
	}
	if result["task_id"] != "task-1" {
		t.Errorf("expected task_id 'task-1', got %q", result["task_id"])
	}
	if result["status"] != "cancelled" {
		t.Errorf("expected status 'cancelled', got %q", result["status"])
	}
	if result["message"] == "" {
		t.Error("expected non-empty message in stop output")
	}
}

func TestBackground_TaskToolsAutoRegisteredWithListTasks(t *testing.T) {
	childLLM := newMockLLM()
	child := agent.New(childLLM)

	var receivedToolNames []string
	var mu sync.Mutex
	parentBase := newMockLLM(mockResponse{Content: "done"})
	parentLLM := &toolCapturingLLM{
		base: parentBase,
		onTools: func(tools []string) {
			mu.Lock()
			receivedToolNames = tools
			mu.Unlock()
		},
	}

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	_, err := parent.Chat(context.Background(), "check tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	var hasListTasks bool
	for _, name := range receivedToolNames {
		if name == "list_tasks" {
			hasListTasks = true
		}
	}

	if !hasListTasks {
		t.Errorf(
			"expected list_tasks tool to be auto-registered, got tools: %v",
			receivedToolNames,
		)
	}
}

func TestSubAgent_MaxTurns(t *testing.T) {
	// Child LLM that keeps requesting tool calls forever (5 responses with tools, then text).
	// Without max_turns, it would loop all 5. With max_turns=2, it stops after 2 tool iterations.
	childLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c1",
					Name:  "echo",
					Input: `{"text":"1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c2",
					Name:  "echo",
					Input: `{"text":"2"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c3",
					Name:  "echo",
					Input: `{"text":"3"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c4",
					Name:  "echo",
					Input: `{"text":"4"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c5",
					Name:  "echo",
					Input: `{"text":"5"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "child done"},
	)
	child := agent.New(childLLM, agent.WithTools(&echoTool{}))

	parentLLM := newMockLLM(
		// Parent calls sub-agent with max_turns: 2
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"loop test","max_turns":2}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "parent done"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test max turns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "parent done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	// With max_turns=2, child should make 3 LLM calls:
	// call 0 → tool call → execute (iteration 0→1)
	// call 1 → tool call → execute (iteration 1→2)
	// call 2 → tool call → iteration(2) >= maxIter(2) → EXIT
	if childLLM.CallCount() != 3 {
		t.Errorf(
			"expected child to make 3 LLM calls with max_turns=2, got %d",
			childLLM.CallCount(),
		)
	}
}

func TestSubAgent_MaxTurnsDefault(t *testing.T) {
	// Without max_turns, child runs to completion (all tool calls + final text)
	childLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c1",
					Name:  "echo",
					Input: `{"text":"1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c2",
					Name:  "echo",
					Input: `{"text":"2"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "child finished naturally"},
	)
	child := agent.New(childLLM, agent.WithTools(&echoTool{}))

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "worker",
					Input: `{"task":"no limit"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "parent done"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "parent done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	// Without max_turns, child completes naturally: 2 tool iterations + 1 final = 3 calls
	if childLLM.CallCount() != 3 {
		t.Errorf(
			"expected child to make 3 LLM calls (natural completion), got %d",
			childLLM.CallCount(),
		)
	}
}

func TestBackground_MaxTurns(t *testing.T) {
	// Same as TestSubAgent_MaxTurns but via background execution
	childLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c1",
					Name:  "echo",
					Input: `{"text":"1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c2",
					Name:  "echo",
					Input: `{"text":"2"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c3",
					Name:  "echo",
					Input: `{"text":"3"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c4",
					Name:  "echo",
					Input: `{"text":"4"}`,
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
					Input: `{"task":"bg loop","max_turns":1,"background":true}`,
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
		mockResponse{Content: "parent done"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "bg max turns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "parent done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	// With max_turns=1, child should make 2 LLM calls:
	// call 0 → tool call → execute (iteration 0→1)
	// call 1 → tool call → iteration(1) >= maxIter(1) → EXIT
	if childLLM.CallCount() != 2 {
		t.Errorf(
			"expected child to make 2 LLM calls with max_turns=1, got %d",
			childLLM.CallCount(),
		)
	}
}

func TestChatOption_DirectCall(t *testing.T) {
	// Test WithMaxTurns directly on Chat() without sub-agents
	llmClient := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c1",
					Name:  "echo",
					Input: `{"text":"1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c2",
					Name:  "echo",
					Input: `{"text":"2"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "c3",
					Name:  "echo",
					Input: `{"text":"3"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "final"},
	)

	a := agent.New(llmClient, agent.WithTools(&echoTool{}))

	resp, err := a.Chat(
		context.Background(),
		"test direct",
		agent.WithMaxTurns(1),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With max_turns=1: call 0 → tool → execute (iteration 0→1), call 1 → tool → EXIT
	if llmClient.CallCount() != 2 {
		t.Errorf(
			"expected 2 LLM calls with WithMaxTurns(1), got %d",
			llmClient.CallCount(),
		)
	}

	// Without max_turns, it would do all 3 tool calls + final = 4 calls
	_ = resp
}

// -- Test helper: blockingMockLLM that respects context cancellation --

type blockingMockLLM struct {
	delay    time.Duration
	fallback *mockLLM
	onCancel func()
}

func (m *blockingMockLLM) SendMessages(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	select {
	case <-time.After(m.delay):
		return m.fallback.SendMessages(ctx, msgs, tools)
	case <-ctx.Done():
		if m.onCancel != nil {
			m.onCancel()
		}
		return nil, ctx.Err()
	}
}

func (m *blockingMockLLM) SendMessagesWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	info *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return m.fallback.SendMessagesWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *blockingMockLLM) StreamResponse(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	go func() {
		defer close(ch)
		select {
		case <-time.After(m.delay):
			for event := range m.fallback.StreamResponse(ctx, msgs, tools) {
				ch <- event
			}
		case <-ctx.Done():
			if m.onCancel != nil {
				m.onCancel()
			}
			ch <- llm.Event{Type: types.EventError, Error: ctx.Err()}
		}
	}()
	return ch
}

func (m *blockingMockLLM) StreamResponseWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	info *schema.StructuredOutputInfo,
) <-chan llm.Event {
	return m.fallback.StreamResponseWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *blockingMockLLM) Model() model.Model {
	return m.fallback.Model()
}

func (m *blockingMockLLM) SupportsStructuredOutput() bool {
	return m.fallback.SupportsStructuredOutput()
}

// -- Test helper: toolCapturingLLM that records tool names --

type toolCapturingLLM struct {
	base    *mockLLM
	onTools func(toolNames []string)
}

func (m *toolCapturingLLM) SendMessages(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	if m.onTools != nil {
		var names []string
		for _, t := range tools {
			names = append(names, t.Info().Name)
		}
		m.onTools(names)
	}
	return m.base.SendMessages(ctx, msgs, tools)
}

func (m *toolCapturingLLM) SendMessagesWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	info *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return m.base.SendMessagesWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *toolCapturingLLM) StreamResponse(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	return m.base.StreamResponse(ctx, msgs, tools)
}

func (m *toolCapturingLLM) StreamResponseWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	info *schema.StructuredOutputInfo,
) <-chan llm.Event {
	return m.base.StreamResponseWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *toolCapturingLLM) Model() model.Model {
	return m.base.Model()
}

func (m *toolCapturingLLM) SupportsStructuredOutput() bool {
	return m.base.SupportsStructuredOutput()
}

// Ensure unused imports are used
var _ = json.Unmarshal
