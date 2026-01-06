// Package postgres provides a PostgreSQL-backed session store for conversation history.
//
// This package stores sessions as JSONB and does not require any PostgreSQL extensions.
// For memory storage with vector search, use the pgvector integration instead.
//
// Example usage:
//
//	import "github.com/joakimcarlsson/ai/integrations/postgres"
//
//	sessionStore, err := postgres.SessionStore(ctx, "postgres://user:pass@localhost/db")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	agent.New(llm, agent.WithSession("conv-1", sessionStore))
package postgres

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// openDB opens a connection to the PostgreSQL database.
func openDB(connString string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
