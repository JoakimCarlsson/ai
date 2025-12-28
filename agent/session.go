package agent

import (
	"context"

	"github.com/joakimcarlsson/ai/message"
)

// Session represents a conversation session that stores message history.
type Session interface {
	ID() string
	GetMessages(ctx context.Context, limit *int) ([]message.Message, error)
	AddMessages(ctx context.Context, msgs []message.Message) error
	PopMessage(ctx context.Context) (*message.Message, error)
	Clear(ctx context.Context) error
}

// SessionStore manages session persistence and retrieval.
type SessionStore interface {
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, id string) (Session, error)
	Load(ctx context.Context, id string) (Session, error)
	Delete(ctx context.Context, id string) error
}
