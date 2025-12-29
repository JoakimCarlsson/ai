package pgvector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/embeddings"
)

const createMemoriesTableSQL = `
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL,
    content TEXT NOT NULL,
    vector vector(%d),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS memories_owner_idx ON memories(owner_id);
`

const createHNSWIndexSQL = `
CREATE INDEX IF NOT EXISTS memories_vector_idx ON memories USING hnsw (vector vector_cosine_ops)
`

type memoryStore struct {
	db          *sql.DB
	embedder    embeddings.Embedding
	idGenerator IDGenerator
}

// MemoryStore creates a new PostgreSQL-backed memory store with pgvector for semantic search.
// It automatically creates the memories table and pgvector extension if they don't exist.
// The vector dimension is determined from the embedder's model configuration.
func MemoryStore(ctx context.Context, connString string, embedder embeddings.Embedding, opts ...Option) (memory.Store, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	db, err := openDB(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	dims := embedder.Model().EmbeddingDims
	if dims == 0 {
		dims = 1536
	}

	createSQL := fmt.Sprintf(createMemoriesTableSQL, dims)
	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create memories table: %w", err)
	}

	db.ExecContext(ctx, createHNSWIndexSQL)

	return &memoryStore{db: db, embedder: embedder, idGenerator: options.idGenerator}, nil
}

func (s *memoryStore) Store(ctx context.Context, id string, fact string, metadata map[string]any) error {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	vectorStr := vectorToString(resp.Embeddings[0])

	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO memories (id, owner_id, content, vector, metadata)
		VALUES ($1, $2, $3, $4::vector, $5)
	`, s.idGenerator(), id, fact, vectorStr, metadataJSON)

	return err
}

func (s *memoryStore) Search(ctx context.Context, id string, query string, limit int) ([]memory.Entry, error) {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	vectorStr := vectorToString(resp.Embeddings[0])

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_id, content, metadata, created_at, 1 - (vector <=> $1::vector) as score
		FROM memories
		WHERE owner_id = $2
		ORDER BY vector <=> $1::vector
		LIMIT $3
	`, vectorStr, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

func (s *memoryStore) GetAll(ctx context.Context, id string, limit int) ([]memory.Entry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_id, content, metadata, created_at, 0 as score
		FROM memories
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

func (s *memoryStore) Delete(ctx context.Context, memoryID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM memories WHERE id = $1", memoryID)
	return err
}

func (s *memoryStore) Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	vectorStr := vectorToString(resp.Embeddings[0])

	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE memories
		SET content = $1, vector = $2::vector, metadata = $3
		WHERE id = $4
	`, fact, vectorStr, metadataJSON, memoryID)

	return err
}

func scanEntries(rows *sql.Rows) ([]memory.Entry, error) {
	var entries []memory.Entry
	for rows.Next() {
		var entry memory.Entry
		var metadataJSON sql.NullString
		var createdAt time.Time

		if err := rows.Scan(&entry.ID, &entry.OwnerID, &entry.Content, &metadataJSON, &createdAt, &entry.Score); err != nil {
			return nil, err
		}

		entry.CreatedAt = createdAt

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &entry.Metadata); err != nil {
				return nil, err
			}
		}

		entries = append(entries, entry)
	}

	if entries == nil {
		entries = []memory.Entry{}
	}

	return entries, rows.Err()
}

func vectorToString(v []float32) string {
	strs := make([]string, len(v))
	for i, f := range v {
		strs[i] = fmt.Sprintf("%f", f)
	}
	return "[" + strings.Join(strs, ",") + "]"
}
