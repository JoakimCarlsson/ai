// Package session provides session storage interfaces and implementations for conversation persistence.
//
// Sessions allow agents to maintain conversation history across multiple interactions.
// The package defines the core [Store] and [Session] interfaces, along with built-in
// implementations for common use cases.
//
// # Built-in Stores
//
// The package provides two built-in session stores:
//
//   - [MemoryStore]: In-memory storage, useful for testing or single-process applications
//   - [FileStore]: File-based storage, persists sessions to disk as JSON files
//
// # Usage with Agent
//
//	store := session.FileStore("./sessions")
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSession("user-123", store),
//	)
//
// # In-Memory Sessions
//
// For testing or ephemeral sessions:
//
//	store := session.MemoryStore()
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSession("test-session", store),
//	)
//
// # Custom Implementations
//
// Implement the [Store] interface for custom backends like PostgreSQL or Redis.
// See the integrations/postgres package for a PostgreSQL implementation.
package session
