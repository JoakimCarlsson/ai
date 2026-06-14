package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/memory/postgres"
	"github.com/joakimcarlsson/ai/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(ctx context.Context, t *testing.T) string {
	pgContainer, err := pgmodule.Run(ctx,
		"postgres:15-alpine",
		pgmodule.WithDatabase("testdb"),
		pgmodule.WithUsername("postgres"),
		pgmodule.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	return connStr
}

func TestPostgresStore_CreateAndLoad(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
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

func TestPostgresStore_ExistsMissing(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPostgresStore_Delete(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
	require.NoError(t, err)

	_, err = store.Create(ctx, "s1")
	require.NoError(t, err)

	err = store.Delete(ctx, "s1")
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "s1")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPostgresSession_AddAndGetMessages(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
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

func TestPostgresSession_GetMessagesWithLimit(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
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

func TestPostgresSession_PopMessage(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
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

func TestPostgresSession_PopMessageEmpty(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	popped, err := s.PopMessage(ctx)
	require.NoError(t, err)
	assert.Nil(t, popped)
}

func TestPostgresSession_Clear(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
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

func TestPostgresSession_PersistsAcrossLoads(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
	require.NoError(t, err)

	s, err := store.Create(ctx, "s1")
	require.NoError(t, err)

	err = s.AddMessages(ctx, []message.Message{
		message.NewUserMessage("persisted"),
	})
	require.NoError(t, err)

	loaded, err := store.Load(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	got, err := loaded.GetMessages(ctx, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "persisted", got[0].Content().Text)
}

func TestPostgresStore_DeleteRemovesSessionMessages(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
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

func TestPostgresSession_ToolCallRoundTrip(t *testing.T) {
	ctx := context.Background()
	connStr := setupPostgres(ctx, t)

	store, err := postgres.SessionStore(ctx, connStr)
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
