// Package pgvector provides a PostgreSQL-backed memory store using pgvector for semantic search.
//
// This package requires the pgvector extension to be available in your PostgreSQL database.
// The extension and tables are created automatically on first use.
//
// Example usage:
//
//	import "github.com/joakimcarlsson/ai/integrations/pgvector"
//
//	memoryStore, err := pgvector.MemoryStore(ctx, "postgres://user:pass@localhost/db", embedder)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	agent.New(llm, agent.WithMemory("alice", memoryStore))
package pgvector

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
