# Session Management

Sessions persist conversation history across multiple `Chat()` calls.

## Setup

```go
import "github.com/joakimcarlsson/ai/agent/session"

myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithSession("conversation-id", session.FileStore("./sessions")),
)
```

## Built-in Stores

```go
// Persistent JSON files
store := session.FileStore("./sessions")

// In-memory (ephemeral, lost on restart)
store := session.MemoryStore()
```

## Database Stores

Ready-to-use stores for production backends:

- [PostgreSQL](../integrations/postgres.md) — `postgres.SessionStore(ctx, connString)`
- [SQLite](../integrations/sqlite.md) — `sqlite.SessionStore(ctx, db)`

## Store Interface

Implement this interface to use any backend:

```go
type Store interface {
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, id string) (Session, error)
    Load(ctx context.Context, id string) (Session, error)
    Delete(ctx context.Context, id string) error
}
```

## Session Interface

```go
type Session interface {
    ID() string
    GetMessages(ctx context.Context, limit *int) ([]message.Message, error)
    AddMessages(ctx context.Context, msgs []message.Message) error
    PopMessage(ctx context.Context) (*message.Message, error)
    Clear(ctx context.Context) error
}
```
