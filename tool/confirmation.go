package tool

import (
	"context"
	"errors"
)

// ErrConfirmationRejected is returned when a tool's confirmation request is rejected by the user.
var ErrConfirmationRejected = errors.New("tool: confirmation rejected")

// ConfirmationRequest carries the details of a tool call that requires human confirmation.
type ConfirmationRequest struct {
	ToolCallID string
	ToolName   string
	Input      string
	Hint       string
	Payload    any
}

// ConfirmationHandler is a function injected into the tool context by the agent layer.
// It blocks until the consumer approves or rejects the request.
// Return nil to approve, or ErrConfirmationRejected to reject.
type ConfirmationHandler func(hint string, payload any) error

type confirmationHandlerKey struct{}

// WithConfirmationHandler returns a new context carrying the given handler.
// The agent layer uses this to inject a handler before calling tool.Run().
func WithConfirmationHandler(
	ctx context.Context,
	handler ConfirmationHandler,
) context.Context {
	return context.WithValue(ctx, confirmationHandlerKey{}, handler)
}

// RequestConfirmation pauses tool execution and asks the consumer for approval.
// Call this from within a tool's Run() method when the tool needs human confirmation
// before proceeding with a sensitive operation.
//
// If no ConfirmationProvider is configured on the agent, this is a no-op (auto-approve).
// If the consumer rejects, ErrConfirmationRejected is returned — the tool should
// propagate this error to halt execution.
//
// Example usage inside a tool's Run():
//
//	if err := tool.RequestConfirmation(ctx, "Delete 42 records from users table", deleteParams); err != nil {
//	    return tool.Response{}, err
//	}
//	// proceed with deletion
func RequestConfirmation(ctx context.Context, hint string, payload any) error {
	handler, ok := ctx.Value(confirmationHandlerKey{}).(ConfirmationHandler)
	if !ok || handler == nil {
		return nil
	}
	return handler(hint, payload)
}
