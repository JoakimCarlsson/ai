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
	Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error
}

type MemoryEntry struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	UserID    string         `json:"user_id"`
	Score     float64        `json:"score"`
	CreatedAt time.Time      `json:"created_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
