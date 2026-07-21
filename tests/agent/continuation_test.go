package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
)

func TestOption_WithContinuationProvider(t *testing.T) {
	called := false
	provider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationDecision, error) {
		called = true
		return agent.ContinuationApprove, nil
	}

	a := agent.New(nil, agent.WithContinuationProvider(provider))
	if a == nil {
		t.Fatal("agent is nil")
	}

	_ = called
}
