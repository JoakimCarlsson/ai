package agent

import (
	"context"

	"github.com/joakimcarlsson/ai/message"
)

type ContinuationDecision string

const (
	ContinuationApprove ContinuationDecision = "approve"
	ContinuationDecline ContinuationDecision = "decline"
	ContinuationTimeout ContinuationDecision = "timeout"
)

type ContinuationRequest struct {
	MaxIterations   int
	TotalIterations int
	ToolCalls       []message.ToolCall
}

type ContinuationResponse struct {
	Decision ContinuationDecision
	Message  string
}

type ContinuationProvider func(ctx context.Context, req ContinuationRequest) (ContinuationResponse, error)
