package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/memory/postgres"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// sharedConnStr is the connection string for a single Postgres container shared
// by every test in the package. Spinning up one container instead of one per
// test keeps the suite fast; tests isolate themselves with unique session IDs.
var sharedConnStr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := pgmodule.Run(ctx,
		"postgres:15-alpine",
		pgmodule.WithDatabase("testdb"),
		pgmodule.WithUsername("postgres"),
		pgmodule.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	sharedConnStr, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get connection string: %v\n", err)
		_ = pgContainer.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()

	if err := pgContainer.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}

	os.Exit(code)
}

// newStore returns a session store backed by the shared container.
func newStore(t *testing.T, opts ...postgres.Option) session.Store {
	t.Helper()
	store, err := postgres.SessionStore(context.Background(), sharedConnStr, opts...)
	require.NoError(t, err)
	return store
}

// sessionID returns a session id unique to the calling test so tests sharing
// the same tables do not interfere with one another.
func sessionID(t *testing.T) string {
	t.Helper()
	return "sess-" + t.Name()
}

func TestPostgresStore_CreateAndLoad(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	id := sessionID(t)

	exists, err := store.Exists(ctx, id)
	require.NoError(t, err)
	assert.False(t, exists)

	s, err := store.Create(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, s.ID())

	exists, err = store.Exists(ctx, id)
	require.NoError(t, err)
	assert.True(t, exists)

	loaded, err := store.Load(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, id, loaded.ID())
}

func TestPostgresStore_ExistsMissing(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	exists, err := store.Exists(ctx, sessionID(t))
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPostgresStore_CreateDuplicateFails(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	id := sessionID(t)

	_, err := store.Create(ctx, id)
	require.NoError(t, err)

	_, err = store.Create(ctx, id)
	require.Error(t, err, "creating a session with a duplicate id should fail")
}

func TestPostgresStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	id := sessionID(t)

	_, err := store.Create(ctx, id)
	require.NoError(t, err)

	err = store.Delete(ctx, id)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, id)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPostgresStore_DeleteMissingIsNoOp(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	err := store.Delete(ctx, sessionID(t))
	require.NoError(t, err)
}

func TestPostgresSession_AddAndGetMessages(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
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

func TestPostgresSession_GetMessagesEmpty(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, got, "expected an empty, non-nil slice")
	assert.Empty(t, got)
}

func TestPostgresSession_AddMessagesEmptyBatch(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	require.NoError(t, s.AddMessages(ctx, nil))
	require.NoError(t, s.AddMessages(ctx, []message.Message{}))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestPostgresSession_GetMessagesWithLimit(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	for i := range 5 {
		err = s.AddMessages(ctx, []message.Message{
			message.NewUserMessage(fmt.Sprintf("msg %d", i)),
		})
		require.NoError(t, err)
		// Small sleep to ensure created_at ordering if the DB resolution is low
		time.Sleep(2 * time.Millisecond)
	}

	limit := 2
	got, err := s.GetMessages(ctx, &limit)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "msg 3", got[0].Content().Text)
	assert.Equal(t, "msg 4", got[1].Content().Text)
}

func TestPostgresSession_GetMessagesLimitExceedsCount(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("a"),
		message.NewUserMessage("b"),
	}))

	limit := 10
	got, err := s.GetMessages(ctx, &limit)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestPostgresSession_PopMessage(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
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

func TestPostgresSession_PopMessageEmpty(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	popped, err := s.PopMessage(ctx)
	require.NoError(t, err)
	assert.Nil(t, popped)
}

func TestPostgresSession_PopMessageDrainsInLIFOOrder(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	for _, text := range []string{"a", "b", "c"} {
		require.NoError(t, s.AddMessages(ctx, []message.Message{
			message.NewUserMessage(text),
		}))
		time.Sleep(2 * time.Millisecond)
	}

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

func TestPostgresSession_Clear(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
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

func TestPostgresSession_ClearThenAdd(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
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

func TestPostgresSession_ClearIsScopedToSession(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s1, err := store.Create(ctx, sessionID(t)+"-1")
	require.NoError(t, err)
	s2, err := store.Create(ctx, sessionID(t)+"-2")
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

func TestPostgresSession_MessagesAreIsolatedBySession(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s1, err := store.Create(ctx, sessionID(t)+"-1")
	require.NoError(t, err)
	s2, err := store.Create(ctx, sessionID(t)+"-2")
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

func TestPostgresSession_PersistsAcrossLoads(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	id := sessionID(t)

	s, err := store.Create(ctx, id)
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("persisted"),
	})
	require.NoError(t, err)

	// A fresh store against the same database must see the message.
	store2 := newStore(t)
	loaded, err := store2.Load(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	got, err := loaded.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "persisted", got[0].Content().Text)
}

func TestPostgresStore_DeleteRemovesSessionMessages(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	id := sessionID(t)

	s, err := store.Create(ctx, id)
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("before delete"),
	})
	require.NoError(t, err)

	err = store.Delete(ctx, id)
	require.NoError(t, err)

	// Recreate the same session ID and verify no old messages remain.
	s, err = store.Create(ctx, id)
	require.NoError(t, err)

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestPostgresStore_DeleteOnlyAffectsTargetSession(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s1, err := store.Create(ctx, sessionID(t)+"-1")
	require.NoError(t, err)
	s2, err := store.Create(ctx, sessionID(t)+"-2")
	require.NoError(t, err)

	require.NoError(t, s1.AddMessages(ctx, []message.Message{
		message.NewUserMessage("s1 message"),
	}))
	require.NoError(t, s2.AddMessages(ctx, []message.Message{
		message.NewUserMessage("s2 message"),
	}))

	require.NoError(t, store.Delete(ctx, sessionID(t)+"-1"))

	exists, err := store.Exists(ctx, sessionID(t)+"-2")
	require.NoError(t, err)
	assert.True(t, exists)

	got, err := s2.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "s2 message", got[0].Content().Text)
}

func TestPostgresSession_ToolCallRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
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

func TestPostgresSession_ToolResultRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
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

func TestPostgresSession_ModelFieldRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	msg := message.NewUserMessage("hello")
	msg.Model = "gpt-4o"
	require.NoError(t, s.AddMessages(ctx, []message.Message{msg}))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "gpt-4o", string(got[0].Model))
}

func TestPostgresSession_UnicodeAndSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	text := "emoji 🚀, quotes \"'`, unicode café ☕, json {\"k\":1}"
	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage(text),
	}))

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, text, got[0].Content().Text)
}

func TestPostgresSession_WithIDGenerator(t *testing.T) {
	ctx := context.Background()

	var counter int
	var mu sync.Mutex
	gen := func() string {
		mu.Lock()
		defer mu.Unlock()
		counter++
		return fmt.Sprintf("%s-msg-%d", t.Name(), counter)
	}

	store := newStore(t, postgres.WithIDGenerator(gen))

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	require.NoError(t, s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("a"),
		message.NewUserMessage("b"),
	}))

	assert.Equal(t, 2, counter, "id generator should be called once per message")

	got, err := s.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestPostgresSession_DuplicateIDGeneratorFails(t *testing.T) {
	ctx := context.Background()

	gen := func() string { return t.Name() + "-constant" }
	store := newStore(t, postgres.WithIDGenerator(gen))

	s, err := store.Create(ctx, sessionID(t))
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("a"),
		message.NewUserMessage("b"),
	})
	require.Error(t, err, "a colliding message id should violate the primary key")
}

func TestPostgresSession_ConcurrentAddMessages(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	s, err := store.Create(ctx, sessionID(t))
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
