package llm

import (
	"context"
	"time"
)

func withTimeout(ctx context.Context, timeout *time.Duration) (context.Context, context.CancelFunc) {
	if timeout != nil {
		return context.WithTimeout(ctx, *timeout)
	}
	return ctx, func() {}
}