package memory

import (
	"context"
	"time"
)

// Store is the interface for memory persistence.
// Users must implement this interface with their own vector store
// (e.g., pgvector, Qdrant, Pinecone) as memory requires embeddings
// for semantic search to work correctly.
type Store interface {
	Store(ctx context.Context, id string, fact string, metadata map[string]any) error
	Search(ctx context.Context, id string, query string, limit int) ([]Entry, error)
	GetAll(ctx context.Context, id string, limit int) ([]Entry, error)
	Delete(ctx context.Context, memoryID string) error
	Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error
}

// Entry represents a single memory entry.
type Entry struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	OwnerID   string         `json:"owner_id"`
	Score     float64        `json:"score"`
	CreatedAt time.Time      `json:"created_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

