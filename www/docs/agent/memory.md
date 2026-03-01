# Persistent Memory

Memory enables cross-conversation fact storage and retrieval using vector-based semantic search.

## Setup

```go
import "github.com/joakimcarlsson/ai/agent/memory"

store := memory.MemoryStore(embedder)

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
store := memory.MemoryStore(embedder)

// File-persisted vector store
store := memory.FileStore("./memories", embedder)
```

## Memory Options

| Option | Description |
|--------|-------------|
| `memory.AutoExtract()` | Automatically extract facts from conversations after each response |
| `memory.AutoDedup()` | Use LLM to deduplicate similar memories before storing |
| `memory.LLM(l)` | Use a separate (cheaper) LLM for extraction and deduplication |

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

When `AutoExtract` is disabled, the agent gets memory tools (`store_memory`, `recall_memories`, `replace_memory`, `delete_memory`) that the LLM can call directly.
