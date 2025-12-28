package agent

import (
	"context"
	"fmt"

	"github.com/joakimcarlsson/ai/message"
)

type Session interface {
	ID() string
	GetMessages(ctx context.Context, limit *int) ([]message.Message, error)
	AddMessages(ctx context.Context, msgs []message.Message) error
	PopMessage(ctx context.Context) (*message.Message, error)
	Clear(ctx context.Context) error
}

type SessionStore interface {
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, id string) (Session, error)
	Load(ctx context.Context, id string) (Session, error)
	Delete(ctx context.Context, id string) error
}

func CreateSession(ctx context.Context, id string, store SessionStore) (Session, error) {
	exists, err := store.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}
	return store.Create(ctx, id)
}

func LoadSession(ctx context.Context, id string, store SessionStore) (Session, error) {
	exists, err := store.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return store.Load(ctx, id)
}

func GetOrCreateSession(ctx context.Context, id string, store SessionStore) (Session, error) {
	exists, err := store.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return store.Load(ctx, id)
	}
	return store.Create(ctx, id)
}
