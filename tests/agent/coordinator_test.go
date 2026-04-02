package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
)

func TestCoordinatorMode_Basic(t *testing.T) {
	leadLLM := newMockLLM(
		mockResponse{Content: "I can only coordinate."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("You are a coordinator."),
		agent.WithTeam(team.Config{Name: "coord-team"}),
		agent.WithCoordinatorMode(),
		agent.WithTools(&echoTool{}),
	)

	resp, err := lead.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "I can only coordinate." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}
