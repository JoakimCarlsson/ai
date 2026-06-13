package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
)

func TestSessionBugs_SystemPromptDuplication(t *testing.T) {
	mock := newMockLLM(
		mockResponse{
			Content:      "Response 1",
			FinishReason: message.FinishReasonEndTurn,
		},
		mockResponse{
			Content:      "Response 2",
			FinishReason: message.FinishReasonEndTurn,
		},
	)

	store := session.MemoryStore()
	ctx := context.Background()

	a := agent.New(mock,
		agent.WithSystemPrompt("You are a test assistant."),
		agent.WithSession("test-sys-dup", store),
	)

	// Turn 1
	_, err := a.Chat(ctx, "Hello")
	if err != nil {
		t.Fatalf("turn 1 failed: %v", err)
	}

	// Turn 2
	_, err = a.Chat(ctx, "How are you?")
	if err != nil {
		t.Fatalf("turn 2 failed: %v", err)
	}

	// Check calls
	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mock.calls))
	}

	// First call should have 1 system message
	countSys := 0
	for _, msg := range mock.calls[0] {
		if msg.Role == message.System {
			countSys++
		}
	}
	if countSys != 1 {
		t.Errorf("call 1: expected 1 system message, got %d", countSys)
	}

	// Second call currently has 2 system messages (Bug 1)
	countSys = 0
	for _, msg := range mock.calls[1] {
		if msg.Role == message.System {
			countSys++
		}
	}
	if countSys != 1 {
		t.Errorf("call 2: expected 1 system message, got %d (Bug 1)", countSys)
	}
}

func TestSessionBugs_StaleSystemPrompt(t *testing.T) {
	mock := newMockLLM(
		mockResponse{
			Content:      "Response 1",
			FinishReason: message.FinishReasonEndTurn,
		},
		mockResponse{
			Content:      "Response 2",
			FinishReason: message.FinishReasonEndTurn,
		},
	)

	store := session.MemoryStore()
	ctx := context.Background()

	// Initial agent with System Prompt A
	a1 := agent.New(mock,
		agent.WithSystemPrompt("Prompt A"),
		agent.WithSession("test-stale-sys", store),
	)

	_, err := a1.Chat(ctx, "Hello")
	if err != nil {
		t.Fatalf("chat 1 failed: %v", err)
	}

	// New agent with same session but System Prompt B
	a2 := agent.New(mock,
		agent.WithSystemPrompt("Prompt B"),
		agent.WithSession("test-stale-sys", store),
	)

	_, err = a2.Chat(ctx, "Hello again")
	if err != nil {
		t.Fatalf("chat 2 failed: %v", err)
	}

	// Check calls
	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mock.calls))
	}

	// Second call should NOT have Prompt A in its history
	for _, msg := range mock.calls[1] {
		if msg.Role == message.System && msg.Content().Text == "Prompt A" {
			t.Errorf("call 2 contains stale Prompt A (Bug 2)")
		}
	}
}
