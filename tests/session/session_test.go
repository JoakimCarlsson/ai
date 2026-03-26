package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
)

func TestMemoryStore_CreateAndLoad(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()

	exists, err := store.Exists(ctx, "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected session to not exist yet")
	}

	s, err := store.Create(ctx, "s1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if s.ID() != "s1" {
		t.Errorf("expected ID 's1', got %q", s.ID())
	}

	exists, err = store.Exists(ctx, "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected session to exist after create")
	}

	loaded, err := store.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil session from Load")
	}
	if loaded.ID() != "s1" {
		t.Errorf("expected ID 's1', got %q", loaded.ID())
	}
}

func TestMemoryStore_LoadMissing(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()

	loaded, err := store.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for missing session")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()

	_, _ = store.Create(ctx, "s1")
	if err := store.Delete(ctx, "s1"); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	exists, _ := store.Exists(ctx, "s1")
	if exists {
		t.Error("expected session gone after delete")
	}
}

func TestMemorySession_AddAndGetMessages(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	s, _ := store.Create(ctx, "s1")

	msgs := []message.Message{
		message.NewUserMessage("hello"),
		message.NewSystemMessage("system"),
	}
	if err := s.AddMessages(ctx, msgs); err != nil {
		t.Fatalf("add error: %v", err)
	}

	got, err := s.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Content().Text != "hello" {
		t.Errorf(
			"expected 'hello', got %q",
			got[0].Content().Text,
		)
	}
}

func TestMemorySession_GetMessagesWithLimit(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	s, _ := store.Create(ctx, "s1")

	for _, text := range []string{"a", "b", "c", "d", "e"} {
		_ = s.AddMessages(
			ctx,
			[]message.Message{message.NewUserMessage(text)},
		)
	}

	limit := 2
	got, err := s.GetMessages(ctx, &limit)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Content().Text != "d" {
		t.Errorf("expected 'd', got %q", got[0].Content().Text)
	}
	if got[1].Content().Text != "e" {
		t.Errorf("expected 'e', got %q", got[1].Content().Text)
	}
}

func TestMemorySession_PopMessage(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	s, _ := store.Create(ctx, "s1")

	_ = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("first"),
		message.NewUserMessage("second"),
	})

	popped, err := s.PopMessage(ctx)
	if err != nil {
		t.Fatalf("pop error: %v", err)
	}
	if popped == nil {
		t.Fatal("expected non-nil popped message")
	}
	if popped.Content().Text != "second" {
		t.Errorf(
			"expected 'second', got %q",
			popped.Content().Text,
		)
	}

	remaining, _ := s.GetMessages(ctx, nil)
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining, got %d", len(remaining))
	}
}

func TestMemorySession_PopMessageEmpty(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	s, _ := store.Create(ctx, "s1")

	popped, err := s.PopMessage(ctx)
	if err != nil {
		t.Fatalf("pop error: %v", err)
	}
	if popped != nil {
		t.Error("expected nil from empty session pop")
	}
}

func TestMemorySession_Clear(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	s, _ := store.Create(ctx, "s1")

	_ = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("hello"),
	})

	if err := s.Clear(ctx); err != nil {
		t.Fatalf("clear error: %v", err)
	}

	got, _ := s.GetMessages(ctx, nil)
	if len(got) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(got))
	}
}

func TestFileStore_CreateAndLoad(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := session.FileStore(dir)
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	s, err := store.Create(ctx, "s1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if s.ID() != "s1" {
		t.Errorf("expected ID 's1', got %q", s.ID())
	}

	exists, _ := store.Exists(ctx, "s1")
	if !exists {
		t.Error("expected session to exist after create")
	}

	if _, err := os.Stat(
		filepath.Join(dir, "s1.json"),
	); err != nil {
		t.Errorf("expected file on disk: %v", err)
	}
}

func TestFileStore_Exists_Missing(t *testing.T) {
	ctx := context.Background()
	store := session.FileStore(t.TempDir())

	exists, err := store.Exists(ctx, "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected false for missing session")
	}
}

func TestFileStore_Delete(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := session.FileStore(dir)

	_, _ = store.Create(ctx, "s1")
	if err := store.Delete(ctx, "s1"); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	exists, _ := store.Exists(ctx, "s1")
	if exists {
		t.Error("expected session gone after delete")
	}
}

func TestFileSession_AddAndGetMessages(t *testing.T) {
	ctx := context.Background()
	store := session.FileStore(t.TempDir())
	s, _ := store.Create(ctx, "s1")

	_ = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("hello"),
		message.NewUserMessage("world"),
	})

	got, err := s.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Content().Text != "hello" {
		t.Errorf(
			"expected 'hello', got %q",
			got[0].Content().Text,
		)
	}
}

func TestFileSession_PersistsAcrossLoads(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := session.FileStore(dir)

	s, _ := store.Create(ctx, "s1")
	_ = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("persisted"),
	})

	loaded, _ := store.Load(ctx, "s1")
	got, err := loaded.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got))
	}
	if got[0].Content().Text != "persisted" {
		t.Errorf(
			"expected 'persisted', got %q",
			got[0].Content().Text,
		)
	}
}

func TestFileSession_GetMessagesWithLimit(t *testing.T) {
	ctx := context.Background()
	store := session.FileStore(t.TempDir())
	s, _ := store.Create(ctx, "s1")

	for _, text := range []string{"a", "b", "c", "d", "e"} {
		_ = s.AddMessages(
			ctx,
			[]message.Message{message.NewUserMessage(text)},
		)
	}

	limit := 3
	got, err := s.GetMessages(ctx, &limit)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got))
	}
	if got[0].Content().Text != "c" {
		t.Errorf("expected 'c', got %q", got[0].Content().Text)
	}
}

func TestFileSession_PopMessage(t *testing.T) {
	ctx := context.Background()
	store := session.FileStore(t.TempDir())
	s, _ := store.Create(ctx, "s1")

	_ = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("first"),
		message.NewUserMessage("second"),
	})

	popped, err := s.PopMessage(ctx)
	if err != nil {
		t.Fatalf("pop error: %v", err)
	}
	if popped.Content().Text != "second" {
		t.Errorf(
			"expected 'second', got %q",
			popped.Content().Text,
		)
	}

	remaining, _ := s.GetMessages(ctx, nil)
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining, got %d", len(remaining))
	}
}

func TestFileSession_PopMessageEmpty(t *testing.T) {
	ctx := context.Background()
	store := session.FileStore(t.TempDir())
	s, _ := store.Create(ctx, "s1")

	popped, err := s.PopMessage(ctx)
	if err != nil {
		t.Fatalf("pop error: %v", err)
	}
	if popped != nil {
		t.Error("expected nil from empty session pop")
	}
}

func TestFileSession_Clear(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := session.FileStore(dir)
	s, _ := store.Create(ctx, "s1")

	_ = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("hello"),
	})

	if err := s.Clear(ctx); err != nil {
		t.Fatalf("clear error: %v", err)
	}

	if _, err := os.Stat(
		filepath.Join(dir, "s1.json"),
	); !os.IsNotExist(err) {
		t.Error("expected file removed after clear")
	}
}

func TestFileSession_ToolCallRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := session.FileStore(t.TempDir())
	s, _ := store.Create(ctx, "s1")

	msg := message.NewMessage(
		message.Assistant,
		[]message.ContentPart{
			message.TextContent{Text: "calling"},
			message.ToolCall{
				ID:    "tc_1",
				Name:  "search",
				Input: `{"q":"test"}`,
			},
		},
	)
	_ = s.AddMessages(ctx, []message.Message{msg})

	got, _ := s.GetMessages(ctx, nil)
	calls := got[0].ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "search" {
		t.Errorf("expected 'search', got %q", calls[0].Name)
	}
}
