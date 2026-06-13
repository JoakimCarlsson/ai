package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
)

func TestSessionBugs_ReSummarization(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	sess, err := store.Create(ctx, "compaction-test")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	msgs := []message.Message{
		message.NewUserMessage("Hello"),
		message.NewAssistantMessage(),
		message.NewUserMessage("Tell me a story"),
		message.NewAssistantMessage(),
	}
	msgs[1].AppendContent("Hi there!")
	msgs[3].AppendContent("Once upon a time...")

	if err := sess.AddMessages(ctx, msgs); err != nil {
		t.Fatalf("failed to add messages: %v", err)
	}

	summary := message.NewSummaryMessage("This is a summary of the conversation so far.")

	// Keep the last 2 messages
	keep := msgs[2:]

	// Use type assertion to call Compact if it's not in the interface yet
	type compacter interface {
		Compact(ctx context.Context, summary message.Message, keep []message.Message) error
	}

	if c, ok := sess.(compacter); ok {
		if err := c.Compact(ctx, summary, keep); err != nil {
			t.Fatalf("failed to compact session: %v", err)
		}
	} else {
		t.Skip("Compact not implemented on session")
	}

	newMsgs, err := sess.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	if len(newMsgs) != 3 {
		t.Errorf("expected 3 messages after compaction, got %d", len(newMsgs))
	}

	if newMsgs[0].Content().Text != summary.Content().Text {
		t.Errorf("expected first message to be summary, got %s", newMsgs[0].Content().Text)
	}

	if newMsgs[1].Content().Text != msgs[2].Content().Text {
		t.Errorf("expected second message to be %s, got %s", msgs[2].Content().Text, newMsgs[1].Content().Text)
	}

	if newMsgs[2].Content().Text != msgs[3].Content().Text {
		t.Errorf("expected third message to be %s, got %s", msgs[3].Content().Text, newMsgs[2].Content().Text)
	}
}

func TestSessionBugs_KeepRecentLost(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	sess, err := store.Create(ctx, "compaction-test-2")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Create a complex sequence with tool calls that MUST be kept together
	msgs := []message.Message{
		message.NewUserMessage("Old message"),
		message.NewAssistantMessage(),
		message.NewUserMessage("Current task"),
		message.NewAssistantMessage(),
		message.NewMessage(message.Tool, []message.ContentPart{
			message.ToolResult{ToolCallID: "call-1", Name: "tool", Content: "Result"},
		}),
		message.NewAssistantMessage(),
	}
	msgs[1].AppendContent("Old response")
	msgs[3].AppendContent("I will help with that")
	msgs[3].AppendToolCalls([]message.ToolCall{
		{ID: "call-1", Name: "tool", Input: "{}"},
	})
	msgs[5].AppendContent("I finished the task")

	if err := sess.AddMessages(ctx, msgs); err != nil {
		t.Fatalf("failed to add messages: %v", err)
	}

	summary := message.NewSummaryMessage("Summary")

	// Keep last 4 messages (the tool call sequence)
	keep := msgs[2:]

	type compacter interface {
		Compact(ctx context.Context, summary message.Message, keep []message.Message) error
	}

	if c, ok := sess.(compacter); ok {
		if err := c.Compact(ctx, summary, keep); err != nil {
			t.Fatalf("failed to compact session: %v", err)
		}
	} else {
		t.Skip("Compact not implemented on session")
	}

	newMsgs, err := sess.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	if len(newMsgs) != 5 { // summary + 4 kept
		t.Errorf("expected 5 messages after compaction, got %d", len(newMsgs))
	}
}
