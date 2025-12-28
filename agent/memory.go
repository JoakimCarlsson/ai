package agent

import (
	"context"
	"time"
)

type Memory interface {
	Store(ctx context.Context, userID string, fact string, metadata map[string]any) error
	Search(ctx context.Context, userID string, query string, limit int) ([]MemoryEntry, error)
	GetAll(ctx context.Context, userID string, limit int) ([]MemoryEntry, error)
	Delete(ctx context.Context, memoryID string) error
}

type MemoryEntry struct {
	ID        string
	Content   string
	UserID    string
	Score     float64
	CreatedAt time.Time
	Metadata  map[string]any
}
