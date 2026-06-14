package session

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
)

func TestMemorySession_Compact(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	s, _ := store.Create(ctx, "s1")

	// Add some initial messages
	msgs := []message.Message{
		message.NewUserMessage("msg 1"),
		message.NewMessage(message.Assistant, []message.ContentPart{message.TextContent{Text: "reply 1"}}),
		message.NewUserMessage("msg 2"),
		message.NewMessage(message.Assistant, []message.ContentPart{message.TextContent{Text: "reply 2"}}),
	}
	if err := s.AddMessages(ctx, msgs); err != nil {
		t.Fatalf("add error: %v", err)
	}

	summary := message.NewSummaryMessage("This is a summary of the first two messages")
	keep := msgs[2:] // Keep "msg 2" and "reply 2"

	if err := s.Compact(ctx, summary, keep); err != nil {
		t.Fatalf("compact error: %v", err)
	}

	got, err := s.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 messages (summary + 2 kept), got %d", len(got))
	}

	if got[0].Role != message.Summary {
		t.Errorf("expected first message to be summary, got %s", got[0].Role)
	}
	if got[1].Content().Text != "msg 2" {
		t.Errorf("expected second message to be 'msg 2', got %q", got[1].Content().Text)
	}
	if got[2].Content().Text != "reply 2" {
		t.Errorf("expected third message to be 'reply 2', got %q", got[2].Content().Text)
	}
}
