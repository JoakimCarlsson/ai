package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

func TestHandoff_Basic(t *testing.T) {
	billingLLM := newMockLLM(mockResponse{Content: "Your bill is $42."})
	billing := agent.New(billingLLM, agent.WithSystemPrompt("You handle billing."))

	triageLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "transfer_to_billing", Input: `{"reason":"billing question"}`, Type: "function"},
			},
		},
	)

	triage := agent.New(triageLLM,
		agent.WithSystemPrompt("You route users to specialists."),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "billing",
			Description: "For billing questions",
			Agent:       billing,
		}),
	)

	resp, err := triage.Chat(context.Background(), "How much do I owe?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Your bill is $42." {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if resp.AgentName != "billing" {
		t.Errorf("expected AgentName 'billing', got %q", resp.AgentName)
	}

	if billingLLM.CallCount() != 1 {
		t.Errorf("expected billing LLM to be called once, got %d", billingLLM.CallCount())
	}
}

func TestHandoff_NoMatch(t *testing.T) {
	billingLLM := newMockLLM()
	billing := agent.New(billingLLM)

	triageLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "some_other_tool", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "handled internally"},
	)

	triage := agent.New(triageLLM,
		agent.WithSystemPrompt("Triage agent"),
		agent.WithTools(&echoTool{}),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "billing",
			Description: "For billing",
			Agent:       billing,
		}),
	)

	resp, err := triage.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "handled internally" {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if resp.AgentName != "" {
		t.Errorf("expected empty AgentName, got %q", resp.AgentName)
	}

	if billingLLM.CallCount() != 0 {
		t.Error("billing LLM should not have been called")
	}
}

func TestHandoff_Chain(t *testing.T) {
	finalLLM := newMockLLM(mockResponse{Content: "final agent response"})
	finalAgent := agent.New(finalLLM, agent.WithSystemPrompt("Final agent"))

	middleLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-2", Name: "transfer_to_final", Input: `{}`, Type: "function"},
			},
		},
	)
	middleAgent := agent.New(middleLLM,
		agent.WithSystemPrompt("Middle agent"),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "final",
			Description: "Final handler",
			Agent:       finalAgent,
		}),
	)

	firstLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "transfer_to_middle", Input: `{}`, Type: "function"},
			},
		},
	)
	firstAgent := agent.New(firstLLM,
		agent.WithSystemPrompt("First agent"),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "middle",
			Description: "Middle handler",
			Agent:       middleAgent,
		}),
	)

	resp, err := firstAgent.Chat(context.Background(), "route me through")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "final agent response" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestHandoff_ReasonOptional(t *testing.T) {
	targetLLM := newMockLLM(mockResponse{Content: "target response"})
	target := agent.New(targetLLM)

	sourceLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "transfer_to_target", Input: `{}`, Type: "function"},
			},
		},
	)
	source := agent.New(sourceLLM,
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "target",
			Description: "Target agent",
			Agent:       target,
		}),
	)

	resp, err := source.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "target response" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestHandoff_Stream(t *testing.T) {
	billingLLM := newMockLLM(mockResponse{Content: "billing response via stream"})
	billing := agent.New(billingLLM, agent.WithSystemPrompt("Billing"))

	triageLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "transfer_to_billing", Input: `{}`, Type: "function"},
			},
		},
	)
	triage := agent.New(triageLLM,
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "billing",
			Description: "Billing",
			Agent:       billing,
		}),
	)

	var gotHandoffEvent bool
	var handoffAgentName string
	var finalContent string

	for event := range triage.ChatStream(context.Background(), "billing question") {
		if event.Type == types.EventHandoff {
			gotHandoffEvent = true
			handoffAgentName = event.AgentName
		}
		if event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if !gotHandoffEvent {
		t.Error("expected EventHandoff event")
	}

	if handoffAgentName != "billing" {
		t.Errorf("expected handoff to 'billing', got %q", handoffAgentName)
	}

	if finalContent != "billing response via stream" {
		t.Errorf("unexpected final content: %q", finalContent)
	}
}

func TestHandoff_AgentNameInStreamResponse(t *testing.T) {
	targetLLM := newMockLLM(mockResponse{Content: "target replied"})
	target := agent.New(targetLLM)

	sourceLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "transfer_to_target", Input: `{}`, Type: "function"},
			},
		},
	)
	source := agent.New(sourceLLM,
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "target",
			Description: "Target",
			Agent:       target,
		}),
	)

	var agentName string
	for event := range source.ChatStream(context.Background(), "test") {
		if event.Response != nil {
			agentName = event.Response.AgentName
		}
	}

	if agentName != "target" {
		t.Errorf("expected AgentName 'target' in response, got %q", agentName)
	}
}
