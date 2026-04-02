package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

func TestCoordinatorMode_SpawnAndCommunicate(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "work done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"w","task":"do work"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "list_teammates",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-3",
					Name:  "read_messages",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "coordinated"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Coordinator"),
		agent.WithTeam(team.Config{Name: "coord-full"}),
		agent.WithCoordinatorMode(),
		agent.WithTools(&echoTool{}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"w": agent.New(teammateLLM,
				agent.WithSystemPrompt("Worker"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "coordinate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "coordinated" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestCoordinatorMode_WithBoardTools(t *testing.T) {
	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "create_board_task",
					Input: `{"title":"Coordinated task"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "list_board_tasks",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "board managed"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Coordinator"),
		agent.WithTeam(team.Config{Name: "coord-board"}),
		agent.WithCoordinatorMode(),
		agent.WithTools(&echoTool{}),
	)

	resp, err := lead.Chat(context.Background(), "manage board")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "board managed" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestTeamStream_AllEventTypes(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "send_message",
					Input: `{"to":"__lead__","content":"status update"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"streamer","task":"stream test"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "all events"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "stream-all"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"streamer": agent.New(teammateLLM,
				agent.WithSystemPrompt("Streamer"),
			),
		}),
	)

	events := map[types.EventType]bool{}
	for event := range lead.ChatStream(
		context.Background(),
		"stream",
	) {
		events[event.Type] = true
	}

	expected := []types.EventType{
		types.EventTeammateSpawned,
		types.EventTeamMessage,
		types.EventTeammateComplete,
		types.EventComplete,
	}
	for _, et := range expected {
		if !events[et] {
			t.Errorf("expected event type %q", et)
		}
	}
}

func TestTeamChat_ContinueWithTeam(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"cont","task":"continue test"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "continued"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "continue-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"cont": agent.New(teammateLLM,
				agent.WithSystemPrompt("Continue"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "continued" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestWithMailbox_NilTeam(t *testing.T) {
	custom := team.NewChannelMailbox()

	leadLLM := newMockLLM(
		mockResponse{Content: "ok"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithMailbox(custom),
	)

	resp, err := lead.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}
