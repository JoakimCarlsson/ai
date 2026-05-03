// Package postgres provides a PostgreSQL-backed session store for the agent package.
//
// This package implements the [session.Store] interface using PostgreSQL for durable
// session persistence. It automatically creates the required tables on initialization.
//
// # Installation
//
// This is a separate Go module to avoid adding database dependencies to the core library:
//
//	go get github.com/joakimcarlsson/ai/integrations/postgres
//
// # Basic Usage
//
//	import "github.com/joakimcarlsson/ai/integrations/postgres"
//
//	store, err := postgres.SessionStore(ctx, "postgres://user:pass@localhost/db?sslmode=disable")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSession("user-123", store),
//	)
//
// # Custom ID Generation
//
// By default, UUIDs are used for message IDs. Use [WithIDGenerator] to provide custom IDs:
//
//	var counter uint64
//
//	snowflakeID := func() string {
//	    ts := time.Now().UnixMilli()
//	    id := atomic.AddUint64(&counter, 1)
//	    return fmt.Sprintf("%d-%d", ts, id)
//	}
//
//	store, err := postgres.SessionStore(ctx, connStr,
//	    postgres.WithIDGenerator(snowflakeID),
//	)
//
// # Database Schema
//
// The package creates two tables:
//
//   - sessions: Stores session metadata (id, created_at)
//   - messages: Stores messages with foreign key to sessions (id, session_id, role, parts, model, created_at)
//
// Messages are stored as JSONB for flexible content part serialization.
package postgres
