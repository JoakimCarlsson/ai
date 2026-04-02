package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

func TestTeamRunner_StreamEvents(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "teammate done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"helper","task":"Help out"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "All done."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "stream-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"helper": agent.New(teammateLLM,
				agent.WithSystemPrompt("Helper"),
			),
		}),
	)

	var sawSpawned bool
	var sawComplete bool

	for event := range lead.ChatStream(
		context.Background(),
		"Go",
	) {
		switch event.Type {
		case types.EventTeammateSpawned:
			sawSpawned = true
		case types.EventTeammateComplete:
			sawComplete = true
		}
	}

	if !sawSpawned {
		t.Error("expected EventTeammateSpawned event")
	}
	if !sawComplete {
		t.Error("expected EventTeammateComplete event")
	}
}

func TestTeamRunner_TeammateCompletion(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "finished work"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"worker","task":"Do work"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "read_messages",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "Got completion notification."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "complete-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"worker": agent.New(teammateLLM,
				agent.WithSystemPrompt("Worker"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "Start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Got completion notification." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}
