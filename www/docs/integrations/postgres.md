# PostgreSQL

PostgreSQL-backed session store for persistent conversation history. No extensions required.

## Installation

```bash
go get github.com/joakimcarlsson/ai/integrations/postgres
```

## Setup

```go
import "github.com/joakimcarlsson/ai/integrations/postgres"

sessionStore, err := postgres.SessionStore(ctx, "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
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
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    parts JSONB NOT NULL,
    model TEXT,
    created_at BIGINT NOT NULL
);

CREATE INDEX messages_session_idx ON messages(session_id, created_at);
```

## Options

| Option | Description |
|--------|-------------|
| `postgres.WithIDGenerator(fn)` | Custom ID generator for message records. Default: UUID v4 |

```go
store, err := postgres.SessionStore(ctx, connString,
    postgres.WithIDGenerator(func() string {
        return myCustomID()
    }),
)
```

## Full Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/agent/memory"
    "github.com/joakimcarlsson/ai/embeddings"
    "github.com/joakimcarlsson/ai/integrations/pgvector"
    "github.com/joakimcarlsson/ai/integrations/postgres"
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
)

func main() {
    ctx := context.Background()
    connString := "postgres://postgres:password@localhost:5432/example?sslmode=disable"

    embedder, err := embeddings.NewEmbedding(
        model.ProviderOpenAI,
        embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
    )
    if err != nil {
        log.Fatal(err)
    }

    llmClient, err := llm.NewLLM(
        model.ProviderOpenAI,
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel(model.OpenAIModels[model.GPT4o]),
    )
    if err != nil {
        log.Fatal(err)
    }

    sessionStore, err := postgres.SessionStore(ctx, connString)
    if err != nil {
        log.Fatal(err)
    }

    memoryStore, err := pgvector.MemoryStore(ctx, connString, embedder)
    if err != nil {
        log.Fatal(err)
    }

    myAgent := agent.New(llmClient,
        agent.WithSystemPrompt("You are a personal assistant with memory."),
        agent.WithSession("conv-1", sessionStore),
        agent.WithMemory("alice", memoryStore,
            memory.AutoExtract(),
            memory.AutoDedup(),
        ),
    )

    response, err := myAgent.Chat(ctx, "Hi! My name is Alice and I love Italian food.")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(response.Content)
}
```
