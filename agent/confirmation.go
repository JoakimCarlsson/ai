package agent

import (
	"context"

	"github.com/joakimcarlsson/ai/tool"
)

// ConfirmationProvider is a callback that decides whether a tool call should proceed.
// It blocks until the consumer provides a decision. Return true to approve, false to reject.
// The provider is called both for tools with RequireConfirmation=true on their Info
// and for tools that call tool.RequestConfirmation() from within Run().
type ConfirmationProvider func(ctx context.Context, req tool.ConfirmationRequest) (bool, error)

type confirmationChanKey struct{}

func withConfirmationChan(
	ctx context.Context,
	ch chan<- ChatEvent,
) context.Context {
	return context.WithValue(ctx, confirmationChanKey{}, ch)
}

func confirmationChanFromContext(ctx context.Context) chan<- ChatEvent {
	ch, _ := ctx.Value(confirmationChanKey{}).(chan<- ChatEvent)
	return ch
}
