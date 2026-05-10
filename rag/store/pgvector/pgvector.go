// Package pgvector provides a PostgreSQL-backed implementation of
// rag.Store using the pgvector extension for cosine-similarity
// search.
//
// # Lifecycle
//
// Schema is owned by versioned migrations embedded in the package.
// Run Migrate(ctx, dsn, dims) once at deploy time as a privileged
// Postgres role; run New(ctx, dsn, dims) from the application as a
// low-privilege role. New does not issue DDL: it reads the
// rag_pgvector_migrations ledger, refuses to start if the schema
// version is older than the code expects, and verifies that the
// embedding column dimensionality matches the configured dims.
//
// # Privileges
//
//   - Migrate needs CREATE EXTENSION (superuser or trusted-extension
//     owner) on first run, and CREATE on the schema for the table
//     and indexes.
//   - New needs SELECT, INSERT, UPDATE, DELETE on the chunks table
//     and SELECT on the rag_pgvector_migrations ledger. No CREATE,
//     no superuser.
//
// # Example
//
//	import (
//	    "github.com/joakimcarlsson/ai/rag"
//	    pgstore "github.com/joakimcarlsson/ai/rag/store/pgvector"
//	)
//
//	// Once at deploy time:
//	if err := pgstore.Migrate(ctx, adminDSN, 1536); err != nil {
//	    log.Fatal(err)
//	}
//
//	// At application start:
//	store, err := pgstore.New(ctx, runtimeDSN, 1536)
//
//	kb := rag.New("docs", embedder, store)
//
// # Plugging into your own migrator
//
// MigrationsFS returns the embedded migration files as an fs.FS so
// you can hand them to golang-migrate, goose, atlas, or whatever your
// organisation already runs. The SQL files contain Go-template
// placeholders ({{.Table}}, {{.Dims}}) that need substituting before
// they execute; Migrate handles this for you, external migrators
// need a render step.
//
// # Dimension changes
//
// pgvector stores dimensionality in the column type (vector(N)).
// Switching embedders to a different dim is a destructive migration:
// add a sibling column, backfill by re-embedding, swap, drop. New
// refuses to start when the configured dim does not match the
// stored column type so the mismatch is caught loud and early.
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

// Option configures the pgvector store.
type Option func(*config)

type config struct {
	table string
}

func defaultConfig() *config {
	return &config{table: DefaultTable}
}

// WithTable overrides the default table name (rag_chunks). Useful
// when running multiple knowledge-base systems in the same database
// or when the table name encodes the embedder version.
func WithTable(name string) Option {
	return func(c *config) {
		c.table = name
	}
}

// New opens a connection to a previously-migrated pgvector database
// and returns a rag.Store. It does NOT issue DDL. It verifies the
// rag_pgvector_migrations ledger is at or above the schema version
// this code expects and that the embedding column is typed
// vector(<dims>); fails fast otherwise. Run Migrate first.
func New(
	ctx context.Context,
	connString string,
	dims int,
	opts ...Option,
) (rag.Store, error) {
	if dims <= 0 {
		return nil, fmt.Errorf("pgvector: dims must be positive, got %d", dims)
	}
	cfg := defaultConfig()
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

	if err := verifySchema(ctx, db, cfg.table, dims); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &store{db: db, table: cfg.table, dims: dims}, nil
}

// verifySchema checks the ledger version and the embedding column
// type, in that order. Returns errors with concrete remediation hints
// (run Migrate, re-embed, etc).
func verifySchema(
	ctx context.Context,
	db *sql.DB,
	table string,
	expectedDims int,
) error {
	ledger := ledgerTableFor(table)
	var ledgerExists bool
	err := db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = current_schema() AND table_name = $1
)`, ledger).Scan(&ledgerExists)
	if err != nil {
		return fmt.Errorf("pgvector: probe ledger: %w", err)
	}
	if !ledgerExists {
		return fmt.Errorf(
			"pgvector: schema not initialised (no %s table); run pgvector.Migrate(ctx, dsn, %d) as a privileged role first",
			ledger, expectedDims,
		)
	}

	current, err := readVersion(ctx, db, table)
	if err != nil {
		return err
	}
	expected := SchemaVersion()
	if current < expected {
		return fmt.Errorf(
			"pgvector: schema is at version %d but this build expects %d; run pgvector.Migrate to upgrade",
			current, expected,
		)
	}
	// current > expected is forward compatibility; allowed but the
	// caller may want to log it. The library does not.

	return verifyEmbeddingDims(ctx, db, table, expectedDims)
}

// verifyEmbeddingDims reads the actual column type from pg_attribute
// and parses the dim out of format_type, e.g. "vector(1536)".
func verifyEmbeddingDims(
	ctx context.Context,
	db *sql.DB,
	table string,
	expectedDims int,
) error {
	var typFmt string
	err := db.QueryRowContext(ctx, `
SELECT format_type(a.atttypid, a.atttypmod)
FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema()
  AND c.relname = $1
  AND a.attname = 'embedding'
`, table).Scan(&typFmt)
	if err == sql.ErrNoRows {
		return fmt.Errorf(
			"pgvector: table %q does not exist; run pgvector.Migrate first",
			table,
		)
	}
	if err != nil {
		return fmt.Errorf("pgvector: read column type for %s.embedding: %w", table, err)
	}

	actualDims, err := parseVectorDims(typFmt)
	if err != nil {
		return fmt.Errorf(
			"pgvector: cannot parse %s.embedding type %q: %w",
			table, typFmt, err,
		)
	}
	if actualDims != expectedDims {
		return fmt.Errorf(
			"pgvector: schema mismatch for %s.embedding: column is %s but code expects vector(%d); a re-embed migration is required",
			table, typFmt, expectedDims,
		)
	}
	return nil
}

// parseVectorDims pulls N out of a pgvector format_type result like
// "vector(1536)".
func parseVectorDims(typFmt string) (int, error) {
	if !strings.HasPrefix(typFmt, "vector(") || !strings.HasSuffix(typFmt, ")") {
		return 0, fmt.Errorf("not a sized vector type: %q", typFmt)
	}
	inner := typFmt[len("vector(") : len(typFmt)-1]
	var n int
	if _, err := fmt.Sscanf(inner, "%d", &n); err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("non-positive dim: %d", n)
	}
	return n, nil
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

// Close releases the underlying database connection pool.
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
