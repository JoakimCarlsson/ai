package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
)

func TestSpawnTeammate_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "invalid json",
			input: `{invalid`,
			want:  "invalid parameters",
		},
		{
			name:  "empty name",
			input: `{"name":"","task":"do stuff"}`,
			want:  "name is required",
		},
		{
			name:  "empty task",
			input: `{"name":"worker","task":""}`,
			want:  "task is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leadLLM := newMockLLM(
				mockResponse{
					ToolCalls: []message.ToolCall{
						{
							ID:    "tc-1",
							Name:  "spawn_teammate",
							Input: tt.input,
							Type:  "function",
						},
					},
				},
				mockResponse{Content: "done"},
			)

			lead := agent.New(leadLLM,
				agent.WithSystemPrompt("Lead"),
				agent.WithTeam(team.Config{Name: "val-team"}),
			)

			resp, err := lead.Chat(
				context.Background(),
				"test",
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Content != "done" {
				t.Errorf(
					"unexpected response: %q",
					resp.Content,
				)
			}
		})
	}
}

func TestSpawnTeammate_MaxCapacity(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "done"},
		mockResponse{Content: "done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"w1","task":"work"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "spawn_teammate",
					Input: `{"name":"w2","task":"more work"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "cap-team", MaxSize: 1}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"w1": agent.New(teammateLLM,
				agent.WithSystemPrompt("W1"),
			),
			"w2": agent.New(teammateLLM,
				agent.WithSystemPrompt("W2"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestSpawnTeammate_DuplicateName(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"dup","task":"first"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "spawn_teammate",
					Input: `{"name":"dup","task":"second"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "dup-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"dup": agent.New(teammateLLM,
				agent.WithSystemPrompt("Dup"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestSendMessage_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid json",
			input: `{invalid`,
		},
		{
			name:  "empty to",
			input: `{"to":"","content":"hello"}`,
		},
		{
			name:  "empty content",
			input: `{"to":"someone","content":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leadLLM := newMockLLM(
				mockResponse{
					ToolCalls: []message.ToolCall{
						{
							ID:    "tc-1",
							Name:  "send_message",
							Input: tt.input,
							Type:  "function",
						},
					},
				},
				mockResponse{Content: "done"},
			)

			lead := agent.New(leadLLM,
				agent.WithSystemPrompt("Lead"),
				agent.WithTeam(team.Config{Name: "msg-val"}),
			)

			resp, err := lead.Chat(
				context.Background(),
				"test",
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Content != "done" {
				t.Errorf(
					"unexpected response: %q",
					resp.Content,
				)
			}
		})
	}
}

func TestSendMessage_Broadcast(t *testing.T) {
	w1LLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "read_messages",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "got it"},
	)

	w2LLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t2",
					Name:  "read_messages",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "got it too"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"w1","task":"wait for msgs"}`,
					Type:  "function",
				},
				{
					ID:    "tc-1b",
					Name:  "spawn_teammate",
					Input: `{"name":"w2","task":"wait for msgs"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "send_message",
					Input: `{"to":"*","content":"hello all"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "broadcasted"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "broadcast-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"w1": agent.New(w1LLM,
				agent.WithSystemPrompt("W1"),
			),
			"w2": agent.New(w2LLM,
				agent.WithSystemPrompt("W2"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "broadcast")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "broadcasted" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestStopTeammate_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid json",
			input: `{invalid`,
		},
		{
			name:  "empty name",
			input: `{"name":""}`,
		},
		{
			name:  "not found",
			input: `{"name":"nonexistent"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leadLLM := newMockLLM(
				mockResponse{
					ToolCalls: []message.ToolCall{
						{
							ID:    "tc-1",
							Name:  "stop_teammate",
							Input: tt.input,
							Type:  "function",
						},
					},
				},
				mockResponse{Content: "done"},
			)

			lead := agent.New(leadLLM,
				agent.WithSystemPrompt("Lead"),
				agent.WithTeam(team.Config{Name: "stop-val"}),
			)

			resp, err := lead.Chat(
				context.Background(),
				"test",
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Content != "done" {
				t.Errorf(
					"unexpected response: %q",
					resp.Content,
				)
			}
		})
	}
}

func TestBoardTools_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    string
	}{
		{
			name:     "create invalid json",
			toolName: "create_board_task",
			input:    `{invalid`,
		},
		{
			name:     "create empty title",
			toolName: "create_board_task",
			input:    `{"title":""}`,
		},
		{
			name:     "claim invalid json",
			toolName: "claim_board_task",
			input:    `{invalid`,
		},
		{
			name:     "claim empty task_id",
			toolName: "claim_board_task",
			input:    `{"task_id":""}`,
		},
		{
			name:     "claim nonexistent task",
			toolName: "claim_board_task",
			input:    `{"task_id":"board-999"}`,
		},
		{
			name:     "complete invalid json",
			toolName: "complete_board_task",
			input:    `{invalid`,
		},
		{
			name:     "complete empty task_id",
			toolName: "complete_board_task",
			input:    `{"task_id":""}`,
		},
		{
			name:     "complete nonexistent task",
			toolName: "complete_board_task",
			input:    `{"task_id":"board-999","result":"x"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leadLLM := newMockLLM(
				mockResponse{
					ToolCalls: []message.ToolCall{
						{
							ID:    "tc-1",
							Name:  tt.toolName,
							Input: tt.input,
							Type:  "function",
						},
					},
				},
				mockResponse{Content: "done"},
			)

			lead := agent.New(leadLLM,
				agent.WithSystemPrompt("Lead"),
				agent.WithTeam(
					team.Config{Name: "board-val"},
				),
			)

			resp, err := lead.Chat(
				context.Background(),
				"test",
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Content != "done" {
				t.Errorf(
					"unexpected response: %q",
					resp.Content,
				)
			}
		})
	}
}

func TestBoardTools_ClaimAlreadyClaimed(t *testing.T) {
	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "create_board_task",
					Input: `{"title":"Task A"}`,
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
					Name:  "claim_board_task",
					Input: `{"task_id":"board-1"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "double-claim"}),
	)

	resp, err := lead.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestBoardTools_CompleteWrongAssignee(t *testing.T) {
	w1LLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "complete_board_task",
					Input: `{"task_id":"board-1","result":"stolen"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "failed"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "create_board_task",
					Input: `{"title":"My task"}`,
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
					Name:  "spawn_teammate",
					Input: `{"name":"thief","task":"Try to complete my task"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "wrong-assignee"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"thief": agent.New(w1LLM,
				agent.WithSystemPrompt("Thief"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "done" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestBoardTools_ListEmpty(t *testing.T) {
	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "list_board_tasks",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "empty"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "empty-board"}),
	)

	resp, err := lead.Chat(context.Background(), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "empty" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestReadMessages_Empty(t *testing.T) {
	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "read_messages",
					Input: `{}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "no messages"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "empty-inbox"}),
	)

	resp, err := lead.Chat(context.Background(), "check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "no messages" {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}
