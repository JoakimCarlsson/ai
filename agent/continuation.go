package agent

import (
	"context"

	"github.com/joakimcarlsson/ai/message"
)

// ContinuationDecision represents a decision on whether to continue an agent loop.
type ContinuationDecision string

const (
	// ContinuationApprove approves the continuation request.
	ContinuationApprove ContinuationDecision = "approve"
	// ContinuationDecline declines the continuation request.
	ContinuationDecline ContinuationDecision = "decline"
	// ContinuationTimeout indicates the continuation request timed out.
	ContinuationTimeout ContinuationDecision = "timeout"
)

// ContinuationRequest represents a request to a ContinuationProvider.
type ContinuationRequest struct {
	MaxIterations   int
	TotalIterations int
	ToolCalls       []message.ToolCall
}

// ContinuationResponse is the response from a ContinuationProvider.
type ContinuationResponse struct {
	Decision         ContinuationDecision
	Message          string
	DiscardToolCalls bool
	ToolMessage      string
}

// ContinuationProvider defines a function that provides continuation decisions.
type ContinuationProvider func(ctx context.Context, req ContinuationRequest) (ContinuationResponse, error)
