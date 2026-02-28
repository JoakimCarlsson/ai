package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
)

func TestSubAgentTool_Run(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Content: "research result about Go"})
	child := agent.New(childLLM, agent.WithSystemPrompt("You are a researcher"))

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "researcher", Input: `{"task":"Research Go programming"}`, Type: "function"},
			},
		},
		mockResponse{Content: "Based on the research: Go is great."},
	)

	parent := agent.New(parentLLM,
		agent.WithSystemPrompt("You coordinate research."),
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "researcher",
			Description: "Researches topics",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "Tell me about Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Based on the research: Go is great." {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if childLLM.CallCount() != 1 {
		t.Errorf("expected child LLM to be called once, got %d", childLLM.CallCount())
	}
}

func TestSubAgentTool_EmptyTask(t *testing.T) {
	childLLM := newMockLLM()
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "researcher", Input: `{"task":""}`, Type: "function"},
			},
		},
		mockResponse{Content: "handled empty task"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "researcher",
			Description: "Researches topics",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if childLLM.CallCount() != 0 {
		t.Error("child should not have been called with empty task")
	}

	if resp.Content != "handled empty task" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestSubAgentTool_InvalidInput(t *testing.T) {
	childLLM := newMockLLM()
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "researcher", Input: `not json`, Type: "function"},
			},
		},
		mockResponse{Content: "handled invalid input"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "researcher",
			Description: "Researches topics",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if childLLM.CallCount() != 0 {
		t.Error("child should not have been called with invalid input")
	}

	if resp.Content != "handled invalid input" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestSubAgentTool_ChildError(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Err: context.DeadlineExceeded})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "researcher", Input: `{"task":"do something"}`, Type: "function"},
			},
		},
		mockResponse{Content: "child failed gracefully"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "researcher",
			Description: "Researches topics",
			Agent:       child,
		}),
	)

	resp, err := parent.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "child failed gracefully" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestSubAgent_MultipleSubAgents(t *testing.T) {
	researchLLM := newMockLLM(mockResponse{Content: "research findings"})
	researcher := agent.New(researchLLM)

	writerLLM := newMockLLM(mockResponse{Content: "written article"})
	writer := agent.New(writerLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "researcher", Input: `{"task":"research AI"}`, Type: "function"},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "writer", Input: `{"task":"write about AI"}`, Type: "function"},
			},
		},
		mockResponse{Content: "final orchestrated result"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(
			agent.SubAgentConfig{Name: "researcher", Description: "Researches topics", Agent: researcher},
			agent.SubAgentConfig{Name: "writer", Description: "Writes articles", Agent: writer},
		),
	)

	resp, err := parent.Chat(context.Background(), "write about AI")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "final orchestrated result" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if researchLLM.CallCount() != 1 {
		t.Errorf("expected researcher to be called once, got %d", researchLLM.CallCount())
	}
	if writerLLM.CallCount() != 1 {
		t.Errorf("expected writer to be called once, got %d", writerLLM.CallCount())
	}
}

func TestSubAgent_Stream(t *testing.T) {
	childLLM := newMockLLM(mockResponse{Content: "streamed child result"})
	child := agent.New(childLLM)

	parentLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "worker", Input: `{"task":"do work"}`, Type: "function"},
			},
		},
		mockResponse{Content: "done with streaming"},
	)

	parent := agent.New(parentLLM,
		agent.WithSubAgents(agent.SubAgentConfig{
			Name:        "worker",
			Description: "Does work",
			Agent:       child,
		}),
	)

	var finalContent string
	for event := range parent.ChatStream(context.Background(), "stream test") {
		if event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "done with streaming" {
		t.Errorf("unexpected final content: %q", finalContent)
	}
}
