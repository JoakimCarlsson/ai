package team

import (
	"context"
	"fmt"
	"testing"

	"github.com/joakimcarlsson/ai/agent/team"
)

func TestChannelMailbox_SendAndRead(t *testing.T) {
	mb := team.NewChannelMailbox()
	defer mb.Close()

	mb.RegisterRecipient("alice")
	mb.RegisterRecipient("bob")
	ctx := context.Background()

	err := mb.Send(ctx, team.Message{
		From: "alice", To: "bob", Content: "hello bob",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs, err := mb.Read(ctx, "bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "hello bob" {
		t.Errorf("expected %q, got %q", "hello bob", msgs[0].Content)
	}
	if msgs[0].From != "alice" {
		t.Errorf("expected from %q, got %q", "alice", msgs[0].From)
	}
	if msgs[0].ID == "" {
		t.Error("expected non-empty message ID")
	}

	msgs, err = mb.Read(ctx, "bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 after drain, got %d", len(msgs))
	}
}

func TestChannelMailbox_Broadcast(t *testing.T) {
	mb := team.NewChannelMailbox()
	defer mb.Close()

	mb.RegisterRecipient("alice")
	mb.RegisterRecipient("bob")
	mb.RegisterRecipient("carol")
	ctx := context.Background()

	err := mb.Send(ctx, team.Message{
		From: "alice", To: "*", Content: "broadcast",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bobMsgs, _ := mb.Read(ctx, "bob")
	carolMsgs, _ := mb.Read(ctx, "carol")
	aliceMsgs, _ := mb.Read(ctx, "alice")

	if len(bobMsgs) != 1 {
		t.Errorf("bob: expected 1, got %d", len(bobMsgs))
	}
	if len(carolMsgs) != 1 {
		t.Errorf("carol: expected 1, got %d", len(carolMsgs))
	}
	if len(aliceMsgs) != 0 {
		t.Errorf("alice (sender): expected 0, got %d", len(aliceMsgs))
	}
}

func TestChannelMailbox_ReadEmpty(t *testing.T) {
	mb := team.NewChannelMailbox()
	defer mb.Close()

	mb.RegisterRecipient("alice")
	ctx := context.Background()

	msgs, err := mb.Read(ctx, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil for empty read, got %v", msgs)
	}
}

func TestChannelMailbox_SendAfterClose(t *testing.T) {
	mb := team.NewChannelMailbox()
	mb.Close()

	err := mb.Send(context.Background(), team.Message{
		From: "a", To: "b", Content: "x",
	})
	if err == nil {
		t.Error("expected error sending to closed mailbox")
	}
}

func TestChannelMailbox_MultipleMessages(t *testing.T) {
	mb := team.NewChannelMailbox()
	defer mb.Close()

	mb.RegisterRecipient("alice")
	mb.RegisterRecipient("bob")
	ctx := context.Background()

	for i := range 5 {
		_ = mb.Send(ctx, team.Message{
			From:    "alice",
			To:      "bob",
			Content: fmt.Sprintf("msg-%d", i),
		})
	}

	msgs, _ := mb.Read(ctx, "bob")
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
	for i, m := range msgs {
		expected := fmt.Sprintf("msg-%d", i)
		if m.Content != expected {
			t.Errorf("message %d: expected %q, got %q", i, expected, m.Content)
		}
	}
}
