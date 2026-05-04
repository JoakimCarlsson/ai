package llm

import (
	"context"
	"time"
)

// ApplyTimeout returns a context with the given timeout applied if non-nil,
// otherwise the original context. The returned cancel func is always safe to call.
func ApplyTimeout(
	ctx context.Context,
	timeout *time.Duration,
) (context.Context, context.CancelFunc) {
	if timeout != nil {
		return context.WithTimeout(ctx, *timeout)
	}
	return ctx, func() {}
}
