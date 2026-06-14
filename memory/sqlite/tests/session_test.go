package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/memory/sqlite"
	"github.com/joakimcarlsson/ai/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// setupSQLite returns an in-memory SQLite database suitable for tests.
//
// database/sql opens a pool of connections and each connection to ":memory:"
// gets its own independent database, which can surface as missing tables or
// empty reads. Capping the pool at a single connection guarantees every
// operation in a test sees the same in-memory database.
func setupSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
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

func TestSQLiteStore_CreateDuplicateFails(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	_, err = store.Create(ctx, "s1")
	require.NoError(t, err)

	_, err = store.Create(ctx, "s1")
	require.Error(t, err, "creating a session with a duplicate id should fail")
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

func TestSQLiteStore_DeleteMissingIsNoOp(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	err = store.Delete(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestSQLiteStore_LoadMissingReturnsUsableSession(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	// Load does not verify existence; it returns a handle bound to the id.
	loaded, err := store.Load(ctx, "never-created")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "never-created", loaded.ID())

	got, err := loaded.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)
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

func TestSQLiteSession_GetMessagesEmpty(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, got, "expected an empty, non-nil slice")
	assert.Empty(t, got)
}

func TestSQLiteSession_AddMessagesEmptyBatch(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	err = s.AddMessages(ctx, nil)
	require.NoError(t, err)
	err = s.AddMessages(ctx, []message.Message{})
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSQLiteSession_AddMessagesPreservesOrder(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	const count = 50
	batch := make([]message.Message, count)
	for i := range batch {
		batch[i] = message.NewUserMessage(fmt.Sprintf("msg %d", i))
	}
	require.NoError(t, s.AddMessages(ctx, batch))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, count)
	for i := range got {
		assert.Equal(t, fmt.Sprintf("msg %d", i), got[i].Content().Text)
	}
}

func TestSQLiteSession_GetMessagesWithLimit(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	for i := range 5 {
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

func TestSQLiteSession_GetMessagesLimitExceedsCount(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("a"),
		message.NewUserMessage("b"),
	}))

	limit := 10
	got, err := s.GetMessages(ctx, &limit)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].Content().Text)
	assert.Equal(t, "b", got[1].Content().Text)
}

func TestSQLiteSession_GetMessagesLimitZero(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("a"),
	}))

	limit := 0
	got, err := s.GetMessages(ctx, &limit)
	require.NoError(t, err)
	assert.Empty(t, got)
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

func TestSQLiteSession_PopMessageDrainsInLIFOOrder(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("a"),
		message.NewUserMessage("b"),
		message.NewUserMessage("c"),
	}))

	for _, want := range []string{"c", "b", "a"} {
		popped, err := s.PopMessage(ctx)
		require.NoError(t, err)
		require.NotNil(t, popped)
		assert.Equal(t, want, popped.Content().Text)
	}

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

func TestSQLiteSession_ClearThenAdd(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("old"),
	}))
	require.NoError(t, s.Clear(ctx))
	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("new"),
	}))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "new", got[0].Content().Text)
}

func TestSQLiteSession_ClearIsScopedToSession(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s1, err := store.Create(ctx, "s1")
	require.NoError(t, err)
	s2, err := store.Create(ctx, "s2")
	require.NoError(t, err)

	require.NoError(t, s1.AddMessages(ctx, []message.Message{
		message.NewUserMessage("from s1"),
	}))
	require.NoError(t, s2.AddMessages(ctx, []message.Message{
		message.NewUserMessage("from s2"),
	}))

	require.NoError(t, s1.Clear(ctx))

	got1, err := s1.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got1)

	got2, err := s2.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got2, 1)
	assert.Equal(t, "from s2", got2[0].Content().Text)
}

func TestSQLiteSession_MessagesAreIsolatedBySession(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s1, err := store.Create(ctx, "s1")
	require.NoError(t, err)
	s2, err := store.Create(ctx, "s2")
	require.NoError(t, err)

	require.NoError(t, s1.AddMessages(ctx, []message.Message{
		message.NewUserMessage("only in s1"),
	}))

	got2, err := s2.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got2)

	got1, err := s1.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got1, 1)
	assert.Equal(t, "only in s1", got1[0].Content().Text)
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

func TestSQLiteStore_DeleteOnlyAffectsTargetSession(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s1, err := store.Create(ctx, "s1")
	require.NoError(t, err)
	s2, err := store.Create(ctx, "s2")
	require.NoError(t, err)

	require.NoError(t, s1.AddMessages(ctx, []message.Message{
		message.NewUserMessage("s1 message"),
	}))
	require.NoError(t, s2.AddMessages(ctx, []message.Message{
		message.NewUserMessage("s2 message"),
	}))

	require.NoError(t, store.Delete(ctx, "s1"))

	exists, err := store.Exists(ctx, "s2")
	require.NoError(t, err)
	assert.True(t, exists)

	got, err := s2.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "s2 message", got[0].Content().Text)
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

func TestSQLiteSession_ToolResultRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	msg := message.NewMessage(
		message.Tool,
		[]message.ContentPart{
			message.ToolResult{
				ToolCallID: "tc_1",
				Name:       "search",
				Content:    `{"result":"ok"}`,
			},
		},
	)
	require.NoError(t, s.AddMessages(ctx, []message.Message{msg}))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)

	results := got[0].ToolResults()
	require.Len(t, results, 1)
	assert.Equal(t, "tc_1", results[0].ToolCallID)
	assert.Equal(t, "search", results[0].Name)
	assert.Equal(t, `{"result":"ok"}`, results[0].Content)
}

func TestSQLiteSession_ModelFieldRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	msg := message.NewUserMessage("hello")
	msg.Model = "gpt-4o"
	require.NoError(t, s.AddMessages(ctx, []message.Message{msg}))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "gpt-4o", string(got[0].Model))
}

func TestSQLiteSession_UnicodeAndSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	text := "emoji 🚀, quotes \"'`, newlines\n\t, unicode café ☕, json {\"k\":1}"
	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage(text),
	}))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, text, got[0].Content().Text)
}

func TestSQLiteStore_Prefix(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store1, err := sqlite.SessionStore(ctx, db, sqlite.WithTablePrefix("a_"))
	require.NoError(t, err)

	store2, err := sqlite.SessionStore(ctx, db, sqlite.WithTablePrefix("b_"))
	require.NoError(t, err)

	_, err = store1.Create(ctx, "s1")
	require.NoError(t, err)

	exists, err := store2.Exists(ctx, "s1")
	require.NoError(t, err)
	assert.False(t, exists, "session in store A should not exist in store B")
}

func TestSQLiteStore_PrefixedStoresShareDB(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	storeA, err := sqlite.SessionStore(ctx, db, sqlite.WithTablePrefix("a_"))
	require.NoError(t, err)
	storeB, err := sqlite.SessionStore(ctx, db, sqlite.WithTablePrefix("b_"))
	require.NoError(t, err)

	sa, err := storeA.Create(ctx, "shared")
	require.NoError(t, err)
	sb, err := storeB.Create(ctx, "shared")
	require.NoError(t, err)

	require.NoError(t, sa.AddMessages(ctx, []message.Message{
		message.NewUserMessage("in A"),
	}))
	require.NoError(t, sb.AddMessages(ctx, []message.Message{
		message.NewUserMessage("in B"),
	}))

	gotA, err := sa.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, gotA, 1)
	assert.Equal(t, "in A", gotA[0].Content().Text)

	gotB, err := sb.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, "in B", gotB[0].Content().Text)

	// Deleting in A must not touch B's identically-named session.
	require.NoError(t, storeA.Delete(ctx, "shared"))
	existsB, err := storeB.Exists(ctx, "shared")
	require.NoError(t, err)
	assert.True(t, existsB)
}

func TestSQLiteSession_ConcurrentAddMessages(t *testing.T) {
	ctx := context.Background()
	db := setupSQLite(t)

	store, err := sqlite.SessionStore(ctx, db)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	const writers = 8
	const perWriter = 10

	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := range writers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range perWriter {
				err := s.AddMessages(ctx, []message.Message{
					message.NewUserMessage(fmt.Sprintf("w%d-m%d", w, i)),
				})
				if err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, got, writers*perWriter)
}
