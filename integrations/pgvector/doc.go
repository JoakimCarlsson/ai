// Package pgvector provides a PostgreSQL + pgvector backed memory store for the agent package.
//
// This package implements the [memory.Store] interface using PostgreSQL with the pgvector
// extension for efficient vector similarity search. It automatically creates the required
// tables and enables the pgvector extension on initialization.
//
// # Prerequisites
//
// PostgreSQL must have the pgvector extension installed:
//
//	CREATE EXTENSION IF NOT EXISTS vector;
//
// # Installation
//
// This is a separate Go module to avoid adding database dependencies to the core library:
//
//	go get github.com/joakimcarlsson/ai/integrations/pgvector
//
// # Basic Usage
//
//	import "github.com/joakimcarlsson/ai/integrations/pgvector"
//
//	embedder, _ := embeddings.NewEmbedding(model.ProviderOpenAI,
//	    embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
//	    embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
//	)
//
//	store, err := pgvector.MemoryStore(ctx, "postgres://user:pass@localhost/db?sslmode=disable", embedder)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	myAgent := agent.New(llmClient,
//	    agent.WithMemory("user-123", store,
//	        memory.AutoExtract(),
//	        memory.AutoDedup(),
//	    ),
//	)
//
// # Custom ID Generation
//
// By default, UUIDs are used for memory entry IDs. Use [WithIDGenerator] to provide custom IDs:
//
//	var counter uint64
//
//	snowflakeID := func() string {
//	    ts := time.Now().UnixMilli()
//	    id := atomic.AddUint64(&counter, 1)
//	    return fmt.Sprintf("%d-%d", ts, id)
//	}
//
//	store, err := pgvector.MemoryStore(ctx, connStr, embedder,
//	    pgvector.WithIDGenerator(snowflakeID),
//	)
//
// # Database Schema
//
// The package creates a memories table with:
//
//   - id: Unique identifier for the memory
//   - owner_id: The memory owner (user/entity ID)
//   - content: The text content of the memory
//   - embedding: Vector embedding for similarity search
//   - metadata: JSONB for additional data
//   - created_at: Timestamp
//
// Similarity search uses cosine distance (<=>) for efficient nearest-neighbor queries.
package pgvector
