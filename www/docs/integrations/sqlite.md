# SQLite

SQLite-backed session store for lightweight persistent conversation history.
Bring your own `*sql.DB` connection with any SQLite driver — the module
itself has zero external SDK deps.

## Installation

```bash
go get github.com/joakimcarlsson/ai/agent/memory/sqlite
```

## Setup

```go
import (
    "database/sql"

    _ "modernc.org/sqlite" // or any SQLite driver
    sqlitemem "github.com/joakimcarlsson/ai/agent/memory/sqlite"
)

db, err := sql.Open("sqlite", "./chat.db")
if err != nil {
    log.Fatal(err)
}

sessionStore, err := sqlitemem.SessionStore(ctx, db)
if err != nil {
    log.Fatal(err)
}

myAgent := agent.New(llmClient,
    agent.WithSession("conv-1", sessionStore),
)
```

Tables and indexes are created automatically on first use.

## Schema

```sql
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL
);

CREATE TABLE messages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role       TEXT NOT NULL,
    parts      TEXT NOT NULL,
    model      TEXT,
    created_at INTEGER NOT NULL
);

CREATE INDEX idx_messages_session ON messages(session_id, id);
```

## Options

| Option | Description |
|---|---|
| `sqlitemem.WithTablePrefix(prefix)` | Prefix for all table names. Useful for multi-tenant or multiple stores in one database |

```go
store, err := sqlitemem.SessionStore(ctx, db,
    sqlitemem.WithTablePrefix("chat_"),
)
// Creates "chat_sessions" and "chat_messages" instead of "sessions" and "messages"
```

## Full example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "os"

    _ "modernc.org/sqlite"

    "github.com/joakimcarlsson/ai/agent"
    sqlitemem "github.com/joakimcarlsson/ai/agent/memory/sqlite"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/model"
)

func main() {
    ctx := context.Background()

    db, err := sql.Open("sqlite", "./chat.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    llmClient := llmopenai.NewLLM(
        llmopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
    )

    sessionStore, err := sqlitemem.SessionStore(ctx, db)
    if err != nil {
        log.Fatal(err)
    }

    myAgent := agent.New(llmClient,
        agent.WithSystemPrompt("You are a helpful assistant."),
        agent.WithSession("conv-1", sessionStore),
    )

    response, err := myAgent.Chat(ctx, "Hello!")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(response.Content)
}
```
