package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

func TestHandoff_Basic(t *testing.T) {
	billingLLM := newMockLLM(mockResponse{Content: "Your bill is $42."})
	billing := agent.New(
		billingLLM,
		agent.WithSystemPrompt("You handle billing."),
	)

	triageLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "transfer_to_billing",
					Input: `{"reason":"billing question"}`,
					Type:  "function",
				},
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
		t.Errorf(
			"expected billing LLM to be called once, got %d",
			billingLLM.CallCount(),
		)
	}
}

func TestHandoff_NoMatch(t *testing.T) {
	billingLLM := newMockLLM()
	billing := agent.New(billingLLM)

	triageLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "some_other_tool",
					Input: `{}`,
					Type:  "function",
				},
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
				{
					ID:    "tc-2",
					Name:  "transfer_to_final",
					Input: `{}`,
					Type:  "function",
				},
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
				{
					ID:    "tc-1",
					Name:  "transfer_to_middle",
					Input: `{}`,
					Type:  "function",
				},
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
				{
					ID:    "tc-1",
					Name:  "transfer_to_target",
					Input: `{}`,
					Type:  "function",
				},
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
	billingLLM := newMockLLM(
		mockResponse{Content: "billing response via stream"},
	)
	billing := agent.New(billingLLM, agent.WithSystemPrompt("Billing"))

	triageLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "transfer_to_billing",
					Input: `{}`,
					Type:  "function",
				},
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
				{
					ID:    "tc-1",
					Name:  "transfer_to_target",
					Input: `{}`,
					Type:  "function",
				},
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

func TestHandoff_MessageHistoryPreserved(t *testing.T) {
	var receivedMsgs []message.Message
	var mu sync.Mutex

	targetBase := newMockLLM(mockResponse{Content: "target response"})
	targetLLM := &toolResultCapturingLLM{
		base: targetBase,
		onCall: func(msgs []message.Message) {
			mu.Lock()
			receivedMsgs = msgs
			mu.Unlock()
		},
	}
	target := agent.New(
		targetLLM,
		agent.WithSystemPrompt("Target system prompt"),
	)

	sourceLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "transfer_to_target",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
	)
	source := agent.New(sourceLLM,
		agent.WithSystemPrompt("Source system prompt"),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "target",
			Description: "Target",
			Agent:       target,
		}),
	)

	_, err := source.Chat(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	var hasUserMsg bool
	for _, msg := range receivedMsgs {
		if msg.Role == message.User {
			txt := msg.Content().Text
			if txt == "hello world" {
				hasUserMsg = true
			}
		}
	}
	if !hasUserMsg {
		t.Error(
			"expected target agent to receive original user message in history",
		)
	}

	var hasAssistant bool
	for _, msg := range receivedMsgs {
		if msg.Role == message.Assistant {
			hasAssistant = true
		}
	}
	if !hasAssistant {
		t.Error(
			"expected target agent to receive assistant messages from source in history",
		)
	}
}

func TestHandoff_SystemPromptSwapped(t *testing.T) {
	var receivedMsgs []message.Message
	var mu sync.Mutex

	targetBase := newMockLLM(mockResponse{Content: "target response"})
	targetLLM := &toolResultCapturingLLM{
		base: targetBase,
		onCall: func(msgs []message.Message) {
			mu.Lock()
			receivedMsgs = msgs
			mu.Unlock()
		},
	}
	target := agent.New(
		targetLLM,
		agent.WithSystemPrompt("I am the TARGET agent"),
	)

	sourceLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "transfer_to_target",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
	)
	source := agent.New(sourceLLM,
		agent.WithSystemPrompt("I am the SOURCE agent"),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "target",
			Description: "Target",
			Agent:       target,
		}),
	)

	_, err := source.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	var systemPrompts []string
	for _, msg := range receivedMsgs {
		if msg.Role == message.System {
			systemPrompts = append(systemPrompts, msg.Content().Text)
		}
	}

	if len(systemPrompts) != 1 {
		t.Fatalf(
			"expected exactly 1 system message after handoff, got %d",
			len(systemPrompts),
		)
	}
	if systemPrompts[0] != "I am the TARGET agent" {
		t.Errorf("expected target system prompt, got %q", systemPrompts[0])
	}
}

func TestHandoff_CircularMaxIterations(t *testing.T) {
	agentALLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-a1",
					Name:  "transfer_to_b",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-a2",
					Name:  "transfer_to_b",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-a3",
					Name:  "transfer_to_b",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "agent A fallback"},
	)

	agentBLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-b1",
					Name:  "transfer_to_a",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-b2",
					Name:  "transfer_to_a",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-b3",
					Name:  "transfer_to_a",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "agent B fallback"},
	)

	agentBStub := agent.New(agentBLLM, agent.WithSystemPrompt("Agent B"))

	agentAFull := agent.New(agentALLM,
		agent.WithSystemPrompt("Agent A"),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "b",
			Description: "Transfer to B",
			Agent:       agentBStub,
		}),
	)

	agentBFull := agent.New(agentBLLM,
		agent.WithSystemPrompt("Agent B"),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "a",
			Description: "Transfer to A",
			Agent:       agentAFull,
		}),
	)

	root := agent.New(agentALLM,
		agent.WithSystemPrompt("Agent A"),
		agent.WithHandoffs(agent.HandoffConfig{
			Name:        "b",
			Description: "Transfer to B",
			Agent:       agentBFull,
		}),
	)

	resp, err := root.Chat(context.Background(), "ping pong")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content == "" && len(resp.ToolCalls) == 0 {
		t.Error("expected either content or pending tool calls")
	}
}
