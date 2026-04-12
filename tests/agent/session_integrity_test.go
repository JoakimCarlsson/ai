package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
)

func validateSessionMessages(
	t *testing.T,
	msgs []message.Message,
) {
	t.Helper()

	for i, msg := range msgs {
		if msg.Role == message.Tool {
			if i == 0 {
				t.Errorf(
					"pos %d: tool message with no preceding assistant",
					i,
				)
				continue
			}
			prev := msgs[i-1]
			if prev.Role != message.Assistant {
				t.Errorf(
					"pos %d: tool message preceded by %s, want assistant",
					i, prev.Role,
				)
			}
			if len(prev.ToolCalls()) == 0 {
				t.Errorf(
					"pos %d: tool message preceded by assistant with no tool_calls",
					i,
				)
			}
		}

		if msg.Role == message.Assistant && len(msg.ToolCalls()) > 0 {
			if i+1 >= len(msgs) {
				t.Errorf(
					"pos %d: assistant with tool_calls at end of session (no tool response)",
					i,
				)
				continue
			}
			next := msgs[i+1]
			if next.Role != message.Tool {
				t.Errorf(
					"pos %d: assistant with tool_calls followed by %s, want tool",
					i, next.Role,
				)
			}
		}
	}
}

func TestSessionIntegrity_ToolCallsHaveMatchingResults(t *testing.T) {
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "call-1",
					Name:  "echo",
					Input: `{"text":"hello"}`,
					Type:  "function",
				},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{
			Content:      "Done. I called echo and got: echo: hello",
			FinishReason: message.FinishReasonEndTurn,
		},
	)

	store := session.MemoryStore()
	ctx := context.Background()

	a := agent.New(mock,
		agent.WithSystemPrompt("You are a test assistant."),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-session", store),
	)

	_, err := a.Chat(ctx, "Call the echo tool with hello")
	if err != nil {
		t.Fatalf("turn 1 failed: %v", err)
	}

	sess, err := store.Load(ctx, "test-session")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}

	msgs, err := sess.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}

	validateSessionMessages(t, msgs)
}

func TestSessionIntegrity_MaxIterationsDoesNotDangle(t *testing.T) {
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "call-1",
					Name:  "echo",
					Input: `{"text":"hello"}`,
					Type:  "function",
				},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "call-2",
					Name:  "echo",
					Input: `{"text":"again"}`,
					Type:  "function",
				},
			},
			FinishReason: message.FinishReasonToolUse,
		},
	)

	store := session.MemoryStore()
	ctx := context.Background()

	a := agent.New(mock,
		agent.WithSystemPrompt("You are a test assistant."),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-session-max", store),
	)

	_, _ = a.Chat(ctx, "Call echo twice", agent.WithMaxTurns(1))

	sess, err := store.Load(ctx, "test-session-max")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}

	msgs, err := sess.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}

	validateSessionMessages(t, msgs)
}

func TestSessionIntegrity_SecondTurnLoadsCleanly(t *testing.T) {
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "call-1",
					Name:  "echo",
					Input: `{"text":"first"}`,
					Type:  "function",
				},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{
			Content:      "First turn done.",
			FinishReason: message.FinishReasonEndTurn,
		},
		mockResponse{
			Content:      "Second turn response.",
			FinishReason: message.FinishReasonEndTurn,
		},
	)

	store := session.MemoryStore()
	ctx := context.Background()

	a := agent.New(mock,
		agent.WithSystemPrompt("You are a test assistant."),
		agent.WithTools(&echoTool{}),
		agent.WithSession("test-session-multi", store),
	)

	resp1, err := a.Chat(ctx, "Call echo with first")
	if err != nil {
		t.Fatalf("turn 1 failed: %v", err)
	}
	_ = resp1

	resp2, err := a.Chat(ctx, "What did we do?")
	if err != nil {
		t.Fatalf("turn 2 failed: %v", err)
	}
	_ = resp2

	sess, err := store.Load(ctx, "test-session-multi")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}

	msgs, err := sess.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}

	validateSessionMessages(t, msgs)
}
