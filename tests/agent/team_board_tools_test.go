package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
)

func TestBoardTools_CreateAndList(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "list_board_tasks",
					Input: `{}`,
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
					Name:  "create_board_task",
					Input: `{"title":"Research topic A"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "spawn_teammate",
					Input: `{"name":"worker","task":"Check the board"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "Board managed."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "board-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"worker": agent.New(teammateLLM,
				agent.WithSystemPrompt("Worker"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "Manage tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Board managed." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestBoardTools_ClaimAndComplete(t *testing.T) {
	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "create_board_task",
					Input: `{"title":"Write docs"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "claim_board_task",
					Input: `{"task_id":"board-1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-3",
					Name:  "complete_board_task",
					Input: `{"task_id":"board-1","result":"Docs written"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "All tasks done."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "claim-team"}),
	)

	resp, err := lead.Chat(context.Background(), "Do tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "All tasks done." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}
