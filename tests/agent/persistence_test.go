package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/tokens"
)

type mockStrategy struct {
	popCount    int
	addMessages []message.Message
}

func (s *mockStrategy) Fit(
	_ context.Context,
	input tokens.StrategyInput,
) (*tokens.StrategyResult, error) {
	return &tokens.StrategyResult{
		Messages: input.Messages,
		SessionUpdate: &tokens.SessionUpdate{
			PopCount:    s.popCount,
			AddMessages: s.addMessages,
		},
	}, nil
}

func TestAgent_PersistenceWithPopCount(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	sess, _ := store.Create(ctx, "test-session")

	// Pre-fill session with some messages
	resp1 := message.NewAssistantMessage()
	resp1.AppendContent("Resp 1")
	sess.AddMessages(ctx, []message.Message{
		message.NewUserMessage("Msg 1"),
		resp1,
		message.NewUserMessage("Msg 2"),
	})

	a := agent.New(newMockLLM(mockResponse{Content: "Mock response"}),
		agent.WithSession("test-session", store),
		agent.WithContextStrategy(&mockStrategy{
			popCount: 2,
			addMessages: []message.Message{
				message.NewSummaryMessage("Summary of 1 and Resp 1"),
				message.NewUserMessage("Msg 2 re-anchored"),
				message.NewUserMessage("Msg 3 re-anchored"),
			},
		}, 10000),
	)

	// This should trigger buildMessages which applies the strategy
	_, err := a.Chat(ctx, "Msg 3")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// We need to get the session from the store again to see updates
	updatedSess, _ := store.Load(ctx, "test-session")
	msgs, _ := updatedSess.GetMessages(ctx, nil)

	// Expected messages:
	// 0: Msg 1
	// 1: Resp 1
	// (Msg 2 and Msg 3 were popped)
	// 2: Summary of 1 and Resp 1
	// 3: Msg 2 re-anchored
	// 4: Msg 3 re-anchored
	// 5: Mock response (from assistant)

	if len(msgs) != 6 {
		t.Errorf("expected 6 messages in session, got %d", len(msgs))
		for i, m := range msgs {
			t.Logf("%d: %s - %s", i, m.Role, m.Content().Text)
		}
	} else {
		if msgs[2].Role != message.Summary {
			t.Errorf("expected msg 2 to be Summary, got %s", msgs[2].Role)
		}
		if msgs[3].Content().Text != "Msg 2 re-anchored" {
			t.Errorf(
				"expected msg 3 to be 'Msg 2 re-anchored', got %s",
				msgs[3].Content().Text,
			)
		}
		if msgs[4].Content().Text != "Msg 3 re-anchored" {
			t.Errorf(
				"expected msg 4 to be 'Msg 3 re-anchored', got %s",
				msgs[4].Content().Text,
			)
		}
	}
}

func TestAgent_PersistenceWithPopCount_Continue(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	sess, _ := store.Create(ctx, "test-session-continue")

	// Pre-fill session with some messages
	resp1 := message.NewAssistantMessage()
	resp1.AppendContent("Resp 1")
	sess.AddMessages(ctx, []message.Message{
		message.NewUserMessage("Msg 1"),
		resp1,
		message.NewUserMessage("Msg 2"),
	})

	a := agent.New(newMockLLM(mockResponse{Content: "Mock response"}),
		agent.WithSession("test-session-continue", store),
		agent.WithContextStrategy(&mockStrategy{
			popCount: 1, // Remove "Msg 2"
			addMessages: []message.Message{
				message.NewSummaryMessage("Summary of 1 and Resp 1"),
				message.NewUserMessage("Msg 2 re-anchored"),
			},
		}, 10000),
	)

	// This should trigger buildContinueMessages which applies the strategy.
	// We need to provide a tool result since Continue requires it.
	_, err := a.Continue(ctx, []message.ToolResult{
		{
			ToolCallID: "call_1",
			Content:    "result",
		},
	})
	if err != nil {
		t.Fatalf("Continue failed: %v", err)
	}

	// We need to get the session from the store again to see updates
	updatedSess, _ := store.Load(ctx, "test-session-continue")
	msgs, _ := updatedSess.GetMessages(ctx, nil)

	// Expected messages:
	// 0: Msg 1
	// 1: Resp 1
	// (Msg 2 was popped)
	// 2: Summary of 1 and Resp 1
	// 3: Msg 2 re-anchored
	// 4: Tool result (added by Continue)
	// 5: Mock response (from assistant, because it's a continue)

	if len(msgs) != 6 {
		t.Errorf("expected 6 messages in session, got %d", len(msgs))
		for i, m := range msgs {
			t.Logf("%d: %s - %s", i, m.Role, m.Content().Text)
		}
	} else {
		if msgs[2].Role != message.Summary {
			t.Errorf("expected msg 2 to be Summary, got %s", msgs[2].Role)
		}
		if msgs[3].Content().Text != "Msg 2 re-anchored" {
			t.Errorf(
				"expected msg 3 to be 'Msg 2 re-anchored', got %s",
				msgs[3].Content().Text,
			)
		}
		if msgs[5].Role != message.Assistant {
			t.Errorf("expected msg 5 to be Assistant, got %s", msgs[5].Role)
		}
	}
}
