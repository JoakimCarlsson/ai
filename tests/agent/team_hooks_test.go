package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
)

func TestTeamHooks_OnTeammateJoinAndComplete(t *testing.T) {
	var joinFired, completeFired bool

	teammateLLM := newMockLLM(
		mockResponse{Content: "done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"hooked","task":"Test hooks"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "Hooks tested."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "hook-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"hooked": agent.New(teammateLLM,
				agent.WithSystemPrompt("Hooked agent"),
			),
		}),
		agent.WithHooks(agent.Hooks{
			OnTeammateJoin: func(
				_ context.Context,
				tc agent.TeammateEventContext,
			) {
				if tc.MemberName == "hooked" {
					joinFired = true
				}
			},
			OnTeammateComplete: func(
				_ context.Context,
				tc agent.TeammateEventContext,
			) {
				if tc.MemberName == "hooked" {
					completeFired = true
				}
			},
		}),
	)

	_, err := lead.Chat(context.Background(), "Test hooks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !joinFired {
		t.Error("expected OnTeammateJoin hook to fire")
	}
	if !completeFired {
		t.Error("expected OnTeammateComplete hook to fire")
	}
}
