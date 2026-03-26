# Persistent Memory

Memory enables cross-conversation fact storage and retrieval using vector-based semantic search.

## Setup

```go
import "github.com/joakimcarlsson/ai/agent/memory"

store := memory.NewStore(embedder)

myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are a personal assistant."),
    agent.WithMemory("user-123", store,
        memory.AutoExtract(),  // Auto-extract facts from conversations
        memory.AutoDedup(),    // LLM-based memory deduplication
    ),
)

response, _ := myAgent.Chat(ctx, "My name is Alice and I'm allergic to peanuts.")
// Agent automatically stores this fact and recalls it in future conversations
```

## Built-in Stores

```go
// In-memory vector store
store := memory.NewStore(embedder)

// File-persisted vector store
store := memory.FileStore("./memories", embedder)
```

## Memory Options

| Option | Description |
|--------|-------------|
| `memory.AutoExtract()` | Automatically extract facts from conversations after each response |
| `memory.AutoDedup()` | Use LLM to deduplicate similar memories before storing |
| `memory.LLM(l)` | Use a separate (cheaper) LLM for extraction and deduplication |

## Database Stores

Ready-to-use stores for production backends:

- [pgvector](../integrations/pgvector.md) — `pgvector.MemoryStore(ctx, connString, embedder)` — PostgreSQL with HNSW vector search

## Store Interface

Implement for any vector database backend:

```go
type Store interface {
    Store(ctx context.Context, id string, fact string, metadata map[string]any) error
    Search(ctx context.Context, id string, query string, limit int) ([]Entry, error)
    GetAll(ctx context.Context, id string, limit int) ([]Entry, error)
    Delete(ctx context.Context, memoryID string) error
    Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error
}
```

## How It Works

When `AutoExtract` is enabled:

1. After the agent responds, it reviews the conversation
2. An LLM extracts factual information worth remembering
3. If `AutoDedup` is enabled, the LLM checks for existing similar memories
4. New facts are stored, duplicates are merged or skipped

## Manual Memory Tools

When `AutoExtract` is disabled, the agent gets four memory tools that the LLM can call directly:

### store_memory

Store a fact about the user for future conversations.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `fact` | string | yes | The fact to remember |
| `category` | string | no | One of: `preference`, `personal`, `health`, `professional`, `other` |

### recall_memories

Search for relevant memories. Returns memory IDs for use with replace/delete.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | What to search for |

### replace_memory

Update an existing memory with corrected or updated information.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `memory_id` | string | yes | ID from `recall_memories` results |
| `fact` | string | yes | The updated fact |
| `category` | string | no | One of: `preference`, `personal`, `health`, `professional`, `other` |

### delete_memory

Remove a memory that is no longer accurate or relevant.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `memory_id` | string | yes | ID from `recall_memories` results |
| `reason` | string | no | Why the memory is being deleted |
