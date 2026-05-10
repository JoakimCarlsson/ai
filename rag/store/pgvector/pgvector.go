// Package pgvector provides a PostgreSQL-backed implementation of
// rag.Store using the pgvector extension for cosine-similarity
// search. Vectors persist across process restarts; the table and
// pgvector extension are created on first use.
//
// Schema:
//
//	CREATE TABLE rag_chunks (
//	    id           TEXT PRIMARY KEY,
//	    kb_id        TEXT NOT NULL,
//	    document_id  TEXT NOT NULL,
//	    content      TEXT NOT NULL,
//	    chunk_index  INT  NOT NULL,
//	    metadata     JSONB,
//	    model        TEXT,
//	    embedding    vector(<dims>),
//	    created_at   TIMESTAMPTZ DEFAULT NOW()
//	);
//
// Example:
//
//	import (
//	    "github.com/joakimcarlsson/ai/rag"
//	    pgstore "github.com/joakimcarlsson/ai/rag/store/pgvector"
//	)
//
//	store, err := pgstore.New(ctx,
//	    "postgres://user:pass@localhost:5432/rag?sslmode=disable",
//	    1536, // text-embedding-3-small dimensionality
//	)
//	if err != nil { log.Fatal(err) }
//
//	kb := rag.New("docs", embedder, store)
package pgvector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/joakimcarlsson/ai/rag"
)

// DefaultTable is the table name used when WithTable is not passed.
const DefaultTable = "rag_chunks"

// Option configures the pgvector store at construction time.
type Option func(*config)

type config struct {
	table string
}

// WithTable overrides the default table name (rag_chunks). Useful
// when running multiple knowledge-base systems in the same database.
func WithTable(name string) Option {
	return func(c *config) {
		c.table = name
	}
}

// New connects to a PostgreSQL database, ensures the pgvector
// extension is enabled, creates the schema if missing, and returns a
// rag.Store backed by it. The dims argument MUST match the embedder
// you plan to use (1536 for text-embedding-3-small, 1024 for
// text-embedding-3-large at default truncation, etc).
func New(
	ctx context.Context,
	connString string,
	dims int,
	opts ...Option,
) (rag.Store, error) {
	if dims <= 0 {
		return nil, fmt.Errorf("pgvector: dims must be positive, got %d", dims)
	}

	cfg := &config{table: DefaultTable}
	for _, opt := range opts {
		opt(cfg)
	}
	if !validTableName(cfg.table) {
		return nil, fmt.Errorf("pgvector: invalid table name %q", cfg.table)
	}

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("pgvector: open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pgvector: ping: %w", err)
	}

	createSQL := fmt.Sprintf(`
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS %[1]s (
    id           TEXT PRIMARY KEY,
    kb_id        TEXT NOT NULL,
    document_id  TEXT NOT NULL,
    content      TEXT NOT NULL,
    chunk_index  INT  NOT NULL,
    metadata     JSONB,
    model        TEXT,
    embedding    vector(%[2]d),
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS %[1]s_kb_idx ON %[1]s (kb_id);
`, cfg.table, dims)

	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pgvector: create schema: %w", err)
	}

	// HNSW index for cosine. Best-effort: the cast for vector_cosine_ops
	// requires pgvector >= 0.5; older versions silently skip.
	indexSQL := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %[1]s_hnsw_idx ON %[1]s USING hnsw (embedding vector_cosine_ops)`,
		cfg.table,
	)
	_, _ = db.ExecContext(ctx, indexSQL)

	return &store{db: db, table: cfg.table, dims: dims}, nil
}

type store struct {
	db    *sql.DB
	table string
	dims  int
}

func (s *store) Upsert(
	ctx context.Context,
	kbID string,
	chunks []rag.EmbeddedChunk,
) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgvector: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(`
INSERT INTO %s (id, kb_id, document_id, content, chunk_index, metadata, model, embedding)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::vector)
ON CONFLICT (id) DO UPDATE SET
    kb_id = EXCLUDED.kb_id,
    document_id = EXCLUDED.document_id,
    content = EXCLUDED.content,
    chunk_index = EXCLUDED.chunk_index,
    metadata = EXCLUDED.metadata,
    model = EXCLUDED.model,
    embedding = EXCLUDED.embedding
`, s.table))
	if err != nil {
		return fmt.Errorf("pgvector: prepare upsert: %w", err)
	}
	defer stmt.Close()

	for _, c := range chunks {
		if len(c.Embedding) != s.dims {
			return fmt.Errorf(
				"pgvector: chunk %s has %d-dim embedding, store expects %d",
				c.ID, len(c.Embedding), s.dims,
			)
		}
		var metaJSON []byte
		if c.Metadata != nil {
			metaJSON, err = json.Marshal(c.Metadata)
			if err != nil {
				return fmt.Errorf("pgvector: marshal metadata: %w", err)
			}
		}
		if _, err := stmt.ExecContext(
			ctx,
			c.ID,
			kbID,
			c.DocumentID,
			c.Content,
			c.Index,
			metaJSON,
			c.Model,
			vectorToString(c.Embedding),
		); err != nil {
			return fmt.Errorf("pgvector: upsert chunk %s: %w", c.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgvector: commit: %w", err)
	}
	return nil
}

func (s *store) Search(
	ctx context.Context,
	kbID string,
	embedding []float32,
	k int,
	opts ...rag.SearchOption,
) ([]rag.Hit, error) {
	if k <= 0 || len(embedding) == 0 {
		return nil, nil
	}
	if len(embedding) != s.dims {
		return nil, fmt.Errorf(
			"pgvector: query has %d-dim embedding, store expects %d",
			len(embedding), s.dims,
		)
	}

	// Drain SearchOptions for forward compat. v1 has no settable
	// fields on SearchConfig but invoking the helper guarantees the
	// option pipeline is wired through every store.
	_ = rag.ApplySearchOptions(opts...)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT id, document_id, content, chunk_index, metadata, 1 - (embedding <=> $1::vector) AS score
FROM %s
WHERE kb_id = $2
ORDER BY embedding <=> $1::vector
LIMIT $3
`, s.table), vectorToString(embedding), kbID, k)
	if err != nil {
		return nil, fmt.Errorf("pgvector: query: %w", err)
	}
	defer rows.Close()

	var hits []rag.Hit
	for rows.Next() {
		var (
			id, docID, content string
			idx                int
			metaJSON           sql.NullString
			score              float64
		)
		if err := rows.Scan(&id, &docID, &content, &idx, &metaJSON, &score); err != nil {
			return nil, fmt.Errorf("pgvector: scan: %w", err)
		}
		hit := rag.Hit{
			Chunk: rag.Chunk{
				ID:         id,
				DocumentID: docID,
				Content:    content,
				Index:      idx,
			},
			Score: score,
		}
		if metaJSON.Valid && metaJSON.String != "" {
			if err := json.Unmarshal([]byte(metaJSON.String), &hit.Metadata); err != nil {
				return nil, fmt.Errorf("pgvector: unmarshal metadata: %w", err)
			}
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

func (s *store) Delete(ctx context.Context, chunkID string) error {
	if chunkID == "" {
		return nil
	}
	_, err := s.db.ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM %s WHERE id = $1", s.table),
		chunkID,
	)
	if err != nil {
		return fmt.Errorf("pgvector: delete: %w", err)
	}
	return nil
}

// Close releases the underlying database connection pool. Optional;
// callers can also leave it to process exit.
func (s *store) Close() error { return s.db.Close() }

// validTableName allows letters, digits, underscores. Strict so the
// table name is safe to interpolate into SQL.
func validTableName(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	for _, r := range name {
		if r == '_' ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func vectorToString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%f", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
