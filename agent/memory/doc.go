// Package memory provides long-term memory storage for AI agents using vector embeddings.
//
// Memory allows agents to remember facts about users across conversations. Unlike sessions
// which store raw conversation history, memory stores semantic facts that can be retrieved
// using similarity search.
//
// # How Memory Works
//
// When configured with [AutoExtract], the agent automatically extracts important facts from
// conversations (e.g., "User's name is Alice", "User prefers Italian food"). These facts
// are stored as vector embeddings and retrieved when relevant to future conversations.
//
// # Built-in Stores
//
// The package provides two built-in stores that require an embeddings client:
//
//   - [MemoryStore]: In-memory storage with vector search
//   - [FileStore]: File-based storage with vector search
//
// For production use, see the integrations/pgvector package for PostgreSQL with pgvector.
//
// # Usage with Agent
//
//	embedder, _ := embeddings.NewEmbedding(model.ProviderOpenAI,
//	    embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
//	    embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
//	)
//
//	store := memory.FileStore("./memories", embedder)
//
//	myAgent := agent.New(llmClient,
//	    agent.WithMemory("user-123", store,
//	        memory.AutoExtract(),
//	        memory.AutoDedup(),
//	    ),
//	)
//
// # Memory Options
//
//   - [AutoExtract]: Automatically extract facts from conversations
//   - [AutoDedup]: Deduplicate similar memories to avoid redundancy
//   - [LLM]: Use a separate LLM for memory operations (extraction/deduplication)
//
// # Custom Implementations
//
// Implement the [Store] interface for custom vector databases like Qdrant, Pinecone, or Weaviate.
package memory
