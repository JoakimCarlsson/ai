package agent

import (
	"context"

	"github.com/joakimcarlsson/ai/message"
)

type Session interface {
	ID() string
	GetMessages(ctx context.Context, limit *int) ([]message.Message, error)
	AddMessages(ctx context.Context, msgs []message.Message) error
	PopMessage(ctx context.Context) (*message.Message, error)
	Clear(ctx context.Context) error
}

