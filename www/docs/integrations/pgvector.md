# pgvector

PostgreSQL-backed memory store using [pgvector](https://github.com/pgvector/pgvector)
for semantic vector search. Stores facts as embeddings and retrieves them
using cosine similarity with HNSW indexing.

## Prerequisites

The pgvector extension must be available in your PostgreSQL instance. The
extension is enabled automatically on first use.

## Installation

```bash
go get github.com/joakimcarlsson/ai/agent/memory/pgvector
```

## Setup

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/agent/memory"
    pgvectormem "github.com/joakimcarlsson/ai/agent/memory/pgvector"
)

memoryStore, err := pgvectormem.MemoryStore(ctx,
    "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
    embedder,
)
if err != nil {
    log.Fatal(err)
}

myAgent := agent.New(llmClient,
    agent.WithMemory("user-123", memoryStore,
        memory.AutoExtract(),
        memory.AutoDedup(),
    ),
)
```

The table, pgvector extension, and HNSW index are created automatically on
first use. The vector dimension is auto-detected from the embedder's model
configuration.

## Schema

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE memories (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL,
    content TEXT NOT NULL,
    vector vector(1536),  -- dimension from embedder
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX memories_owner_idx ON memories(owner_id);
CREATE INDEX memories_vector_idx ON memories USING hnsw (vector vector_cosine_ops);
```

## Options

| Option | Description |
|---|---|
| `pgvectormem.WithIDGenerator(fn)` | Custom ID generator for memory records. Default: UUID v4 |

```go
store, err := pgvectormem.MemoryStore(ctx, connString, embedder,
    pgvectormem.WithIDGenerator(func() string {
        return myCustomID()
    }),
)
```

## Full example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/agent/memory"
    pgvectormem "github.com/joakimcarlsson/ai/agent/memory/pgvector"
    pgsessmem "github.com/joakimcarlsson/ai/agent/memory/postgres"
    embopenai "github.com/joakimcarlsson/ai/embeddings/openai"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/model"
)

func main() {
    ctx := context.Background()
    connString := "postgres://postgres:password@localhost:5432/example?sslmode=disable"

    embedder := embopenai.NewEmbedding(
        embopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        embopenai.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
    )

    llmClient := llmopenai.NewLLM(
        llmopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
    )

    // PostgreSQL sessions + pgvector memory
    sessionStore, err := pgsessmem.SessionStore(ctx, connString)
    if err != nil {
        log.Fatal(err)
    }

    memoryStore, err := pgvectormem.MemoryStore(ctx, connString, embedder)
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

    // First conversation — agent learns facts.
    response, err := myAgent.Chat(ctx, "Hi! My name is Alice and I love Italian food.")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(response.Content)

    // New conversation — agent recalls memories via vector search.
    agent2 := agent.New(llmClient,
        agent.WithSystemPrompt("You are a personal assistant with memory."),
        agent.WithSession("conv-2", sessionStore),
        agent.WithMemory("alice", memoryStore,
            memory.AutoExtract(),
            memory.AutoDedup(),
        ),
    )

    response, err = agent2.Chat(ctx, "Can you recommend a restaurant for me?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(response.Content)
}
```
