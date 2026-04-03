package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
)

func TestSpawnTeammate_Basic(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "research complete"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"researcher","task":"Research Go concurrency"}`,
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
		mockResponse{Content: "Team work done."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("You are a team lead."),
		agent.WithTeam(team.Config{
			Name:    "test-team",
			MaxSize: 3,
		}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"researcher": agent.New(teammateLLM,
				agent.WithSystemPrompt("You are a researcher."),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "Start research")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Team work done." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestSpawnTeammate_DynamicSystemPrompt(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "dynamic teammate done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:   "tc-1",
					Name: "spawn_teammate",
					Input: `{"name":"dynamic","task":"Do dynamic work",` +
						`"system_prompt":"You are a dynamic agent."}`,
					Type: "function",
				},
			},
		},
		mockResponse{Content: "Done."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "dyn-team"}),
	)
	_ = teammateLLM

	resp, err := lead.Chat(context.Background(), "Start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Done." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestSendAndReadMessages(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "send_message",
					Input: `{"to":"__lead__","content":"found results"}`,
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
		mockResponse{Content: "Got the results."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("You are a team lead."),
		agent.WithTeam(team.Config{Name: "msg-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"worker": agent.New(teammateLLM,
				agent.WithSystemPrompt("You are a worker."),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "Start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Got the results." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestListTeammates(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"worker","task":"Work"}`,
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
		mockResponse{Content: "Listed."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "list-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"worker": agent.New(teammateLLM,
				agent.WithSystemPrompt("Worker"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "List")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Listed." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}

func TestWithMailbox_Custom(t *testing.T) {
	var sent []team.Message

	custom := &recordingMailbox{
		inner: team.NewChannelMailbox(),
		sent:  &sent,
	}

	teammateLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "send_message",
					Input: `{"to":"__lead__","content":"custom mailbox works"}`,
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
					Input: `{"name":"worker","task":"Test custom mailbox"}`,
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
		mockResponse{Content: "Done."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "custom-mb-team"}),
		agent.WithMailbox(custom),
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

	if resp.Content != "Done." {
		t.Errorf("unexpected response: %q", resp.Content)
	}

	if len(sent) == 0 {
		t.Fatal("expected custom mailbox to record sent messages")
	}

	found := false
	for _, msg := range sent {
		if msg.Content == "custom mailbox works" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'custom mailbox works' in recorded messages")
	}
}

type recordingMailbox struct {
	inner team.Mailbox
	sent  *[]team.Message
}

func (r *recordingMailbox) Send(ctx context.Context, msg team.Message) error {
	*r.sent = append(*r.sent, msg)
	return r.inner.Send(ctx, msg)
}

func (r *recordingMailbox) Read(
	ctx context.Context,
	recipient string,
) ([]team.Message, error) {
	return r.inner.Read(ctx, recipient)
}

func (r *recordingMailbox) RegisterRecipient(name string) {
	r.inner.RegisterRecipient(name)
}

func (r *recordingMailbox) Close() error {
	return r.inner.Close()
}

func TestStopTeammate(t *testing.T) {
	teammateLLM := newMockLLM(
		mockResponse{Content: "this takes a while"},
		mockResponse{Content: "still working"},
		mockResponse{Content: "done"},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"slow","task":"Slow task"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-2",
					Name:  "stop_teammate",
					Input: `{"name":"slow"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "Stopped."},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "stop-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"slow": agent.New(teammateLLM,
				agent.WithSystemPrompt("Slow worker"),
			),
		}),
	)

	resp, err := lead.Chat(context.Background(), "Stop it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Stopped." {
		t.Errorf("unexpected response: %q", resp.Content)
	}
}
