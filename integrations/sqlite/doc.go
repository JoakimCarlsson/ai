// Package sqlite provides a SQLite-backed session store for the agent package.
//
// This package implements the [session.Store] interface using SQLite for durable
// session persistence. It automatically creates the required tables on initialization.
//
// # Installation
//
// This is a separate Go module to avoid adding database dependencies to the core library:
//
//	go get github.com/joakimcarlsson/ai/integrations/sqlite
//
// # Basic Usage
//
// The package accepts an existing *sql.DB connection, allowing the caller to choose
// their preferred SQLite driver and configure connection settings:
//
//	import "github.com/joakimcarlsson/ai/integrations/sqlite"
//
//	store, err := sqlite.SessionStore(ctx, db)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSession("user-123", store),
//	)
//
// # Table Prefix
//
// Use [WithTablePrefix] to namespace tables and avoid conflicts with existing schemas:
//
//	store, err := sqlite.SessionStore(ctx, db,
//	    sqlite.WithTablePrefix("chat_"),
//	)
//
// This creates "chat_sessions" and "chat_messages" tables instead of the default
// "sessions" and "messages".
//
// # Database Schema
//
// The package creates two tables:
//
//   - sessions: Stores session metadata (id, created_at)
//   - messages: Stores messages with foreign key to sessions (id, session_id, role, parts, model, created_at)
//
// Messages are stored as JSON text for flexible content part serialization.
package sqlite
