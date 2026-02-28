package agent

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
)

func TestFanOut_Basic(t *testing.T) {
	workerLLM := newMockLLM(
		mockResponse{Content: "result for task 1"},
		mockResponse{Content: "result for task 2"},
		mockResponse{Content: "result for task 3"},
	)
	worker := agent.New(workerLLM, agent.WithSystemPrompt("Worker"))

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "parallel_work", Input: `{"tasks":["task 1","task 2","task 3"]}`, Type: "function"},
			},
		},
		mockResponse{Content: "all tasks done"},
	)

	parent := agent.New(parentLLM,
		agent.WithFanOut(agent.FanOutConfig{
			Name:        "parallel_work",
			Description: "Runs tasks in parallel",
			Agent:       worker,
		}),
	)

	resp, err := parent.Chat(context.Background(), "do three things")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "all tasks done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if workerLLM.CallCount() != 3 {
		t.Errorf("expected worker to be called 3 times, got %d", workerLLM.CallCount())
	}
}

func TestFanOut_EmptyTasks(t *testing.T) {
	workerLLM := newMockLLM()
	worker := agent.New(workerLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "parallel_work", Input: `{"tasks":[]}`, Type: "function"},
			},
		},
		mockResponse{Content: "handled empty"},
	)

	parent := agent.New(parentLLM,
		agent.WithFanOut(agent.FanOutConfig{
			Name:        "parallel_work",
			Description: "Runs tasks in parallel",
			Agent:       worker,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workerLLM.CallCount() != 0 {
		t.Error("worker should not have been called with empty tasks")
	}

	if resp.Content != "handled empty" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestFanOut_InvalidInput(t *testing.T) {
	workerLLM := newMockLLM()
	worker := agent.New(workerLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "parallel_work", Input: `bad json`, Type: "function"},
			},
		},
		mockResponse{Content: "handled bad input"},
	)

	parent := agent.New(parentLLM,
		agent.WithFanOut(agent.FanOutConfig{
			Name:        "parallel_work",
			Description: "Runs tasks in parallel",
			Agent:       worker,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "handled bad input" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestFanOut_PartialFailure(t *testing.T) {
	workerLLM := newMockLLM(
		mockResponse{Content: "success result"},
		mockResponse{Err: context.DeadlineExceeded},
	)
	worker := agent.New(workerLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "parallel_work", Input: `{"tasks":["succeed","fail"]}`, Type: "function"},
			},
		},
		mockResponse{Content: "partial results handled"},
	)

	parent := agent.New(parentLLM,
		agent.WithFanOut(agent.FanOutConfig{
			Name:        "parallel_work",
			Description: "Runs tasks in parallel",
			Agent:       worker,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "partial results handled" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestFanOut_Concurrency(t *testing.T) {
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	slowLLM := &concurrencyTrackingLLM{
		base:              newMockLLM(),
		maxConcurrent:     &maxConcurrent,
		currentConcurrent: &currentConcurrent,
		delay:             50 * time.Millisecond,
	}
	worker := agent.New(slowLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "parallel_work", Input: `{"tasks":["a","b","c","d","e"]}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	parent := agent.New(parentLLM,
		agent.WithFanOut(agent.FanOutConfig{
			Name:           "parallel_work",
			Description:    "Runs tasks in parallel",
			Agent:          worker,
			MaxConcurrency: 2,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test concurrency")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if maxConcurrent.Load() > 2 {
		t.Errorf("max concurrency exceeded: got %d, expected <= 2", maxConcurrent.Load())
	}
}

func TestFanOut_ResultsAggregated(t *testing.T) {
	workerLLM := newMockLLM(
		mockResponse{Content: "alpha result"},
		mockResponse{Content: "beta result"},
	)
	worker := agent.New(workerLLM)

	var capturedToolResult string
	parentLLM := &toolResultCapturingLLM{
		base: newMockLLM(
			mockResponse{
				ToolCalls: []message.ToolCall{
					{ID: "tc-1", Name: "parallel_work", Input: `{"tasks":["alpha","beta"]}`, Type: "function"},
				},
			},
			mockResponse{Content: "done"},
		),
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
		agent.WithFanOut(agent.FanOutConfig{
			Name:        "parallel_work",
			Description: "Parallel work",
			Agent:       worker,
		}),
	)

	_, err := parent.Chat(context.Background(), "test aggregation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedToolResult, "alpha") {
		t.Error("expected aggregated result to contain 'alpha'")
	}
	if !strings.Contains(capturedToolResult, "beta") {
		t.Error("expected aggregated result to contain 'beta'")
	}
}

func TestFanOut_Stream(t *testing.T) {
	workerLLM := newMockLLM(
		mockResponse{Content: "parallel result 1"},
		mockResponse{Content: "parallel result 2"},
	)
	worker := agent.New(workerLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "fan", Input: `{"tasks":["t1","t2"]}`, Type: "function"},
			},
		},
		mockResponse{Content: "fan-out streamed"},
	)

	parent := agent.New(parentLLM,
		agent.WithFanOut(agent.FanOutConfig{
			Name:        "fan",
			Description: "Fan out",
			Agent:       worker,
		}),
	)

	var finalContent string
	for event := range parent.ChatStream(context.Background(), "stream fan-out") {
		if event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "fan-out streamed" {
		t.Errorf("unexpected final content: %q", finalContent)
	}
}
