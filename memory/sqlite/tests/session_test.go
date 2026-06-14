package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/joakimcarlsson/ai/memory/sqlite"
	"github.com/joakimcarlsson/ai/message"
	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSQLite(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteStore_CreateAndLoad(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "s1")
	require.NoError(t, err)
	assert.False(t, exists)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, "s1", s.ID())

	exists, err = store.Exists(ctx, "s1")
	require.NoError(t, err)
	assert.True(t, exists)

	loaded, err := store.Load(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "s1", loaded.ID())
}

func TestSQLiteStore_ExistsMissing(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSQLiteStore_Delete(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	_, err = store.Create(ctx, "s1")
	require.NoError(t, err)

	err = store.Delete(ctx, "s1")
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "s1")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSQLiteSession_AddAndGetMessages(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	msgs := []message.Message{
		message.NewUserMessage("hello"),
		message.NewSystemMessage("system prompt"),
	}
	err = s.AddMessages(ctx, msgs)
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "hello", got[0].Content().Text)
	assert.Equal(t, message.User, got[0].Role)
	assert.Equal(t, "system prompt", got[1].Content().Text)
	assert.Equal(t, message.System, got[1].Role)
}

func TestSQLiteSession_GetMessagesWithLimit(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = s.AddMessages(ctx, []message.Message{
			message.NewUserMessage(fmt.Sprintf("msg %d", i)),
		})
		require.NoError(t, err)
	}

	limit := 2
	got, err := s.GetMessages(ctx, &limit)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "msg 3", got[0].Content().Text)
	assert.Equal(t, "msg 4", got[1].Content().Text)
}

func TestSQLiteSession_PopMessage(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("first"),
		message.NewUserMessage("second"),
	})
	require.NoError(t, err)

	popped, err := s.PopMessage(ctx)
	require.NoError(t, err)
	require.NotNil(t, popped)
	assert.Equal(t, "second", popped.Content().Text)

	remaining, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, "first", remaining[0].Content().Text)
}

func TestSQLiteSession_PopMessageEmpty(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	popped, err := s.PopMessage(ctx)
	require.NoError(t, err)
	assert.Nil(t, popped)
}

func TestSQLiteSession_Clear(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("hello"),
	})
	require.NoError(t, err)

	err = s.Clear(ctx)
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSQLiteSession_PersistsAcrossLoads(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initial setup
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	
	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("persisted"),
	})
	require.NoError(t, err)
	db.Close()

	// Reload from the same file
	db2, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db2.Close()

	store2, err := sqlite.SessionStore(ctx, db2)
	require.NoError(t, err)

	loaded, err := store2.Load(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	got, err := loaded.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "persisted", got[0].Content().Text)
}

func TestSQLiteStore_DeleteRemovesSessionMessages(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("before delete"),
	})
	require.NoError(t, err)

	err = store.Delete(ctx, "s1")
	require.NoError(t, err)

	// Recreate the same session ID and verify no old messages remain.
	s, err = store.Create(ctx, "s1")
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSQLiteSession_ToolCallRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

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
	err = s.AddMessages(ctx, []message.Message{msg})
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)

	calls := got[0].ToolCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "search", calls[0].Name)
	assert.Equal(t, `{"q":"test"}`, calls[0].Input)
}

func TestSQLiteStore_Prefix(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store1, err := sqlite.SessionStore(ctx, db, sqlite.WithTablePrefix("a_"))
	require.NoError(t, err)

	store2, err := sqlite.SessionStore(ctx, db, sqlite.WithTablePrefix("b_"))
	require.NoError(t, err)

	_, _ = store1.Create(ctx, "s1")
	exists, _ := store2.Exists(ctx, "s1")
	assert.False(t, exists, "session in store A should not exist in store B")
}
