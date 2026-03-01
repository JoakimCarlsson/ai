# pgvector

PostgreSQL-backed memory store using [pgvector](https://github.com/pgvector/pgvector) for semantic vector search. Stores facts as embeddings and retrieves them using cosine similarity with HNSW indexing.

## Prerequisites

pgvector extension must be available in your PostgreSQL instance. The extension is enabled automatically on first use.

## Installation

```bash
go get github.com/joakimcarlsson/ai/integrations/pgvector
```

## Setup

```go
import (
    "github.com/joakimcarlsson/ai/integrations/pgvector"
    "github.com/joakimcarlsson/ai/agent/memory"
)

memoryStore, err := pgvector.MemoryStore(ctx, "postgres://user:pass@localhost:5432/mydb?sslmode=disable", embedder)
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

The table, pgvector extension, and HNSW index are created automatically on first use. The vector dimension is auto-detected from the embedder's model configuration.

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
|--------|-------------|
| `pgvector.WithIDGenerator(fn)` | Custom ID generator for memory records. Default: UUID v4 |

```go
store, err := pgvector.MemoryStore(ctx, connString, embedder,
    pgvector.WithIDGenerator(func() string {
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

    // PostgreSQL sessions + pgvector memory
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

    // First conversation — agent learns facts
    response, err := myAgent.Chat(ctx, "Hi! My name is Alice and I love Italian food.")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(response.Content)

    // New conversation — agent recalls memories via vector search
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
