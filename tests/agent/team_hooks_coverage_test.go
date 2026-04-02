package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

func TestTeamHooks_AllCallbacks(t *testing.T) {
	var mu sync.Mutex
	fired := map[string]bool{}

	teammateLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "send_message",
					Input: `{"to":"__lead__","content":"hi"}`,
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
					Input: `{"name":"worker","task":"say hi"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "all-hooks"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"worker": agent.New(teammateLLM,
				agent.WithSystemPrompt("Worker"),
			),
		}),
		agent.WithHooks(agent.Hooks{
			OnTeammateJoin: func(
				_ context.Context,
				_ agent.TeammateEventContext,
			) {
				mu.Lock()
				fired["join"] = true
				mu.Unlock()
			},
			OnTeammateLeave: func(
				_ context.Context,
				_ agent.TeammateEventContext,
			) {
				mu.Lock()
				fired["leave"] = true
				mu.Unlock()
			},
			OnTeammateComplete: func(
				_ context.Context,
				_ agent.TeammateEventContext,
			) {
				mu.Lock()
				fired["complete"] = true
				mu.Unlock()
			},
			OnTeamMessage: func(
				_ context.Context,
				_ agent.TeamMessageContext,
			) {
				mu.Lock()
				fired["message"] = true
				mu.Unlock()
			},
		}),
	)

	_, err := lead.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	for _, key := range []string{
		"join",
		"complete",
		"message",
	} {
		if !fired[key] {
			t.Errorf("expected %s hook to fire", key)
		}
	}
}

func TestTeamHooks_ErrorCallback(t *testing.T) {
	var mu sync.Mutex
	var errorFired bool
	var capturedErr string

	failLLM := newMockLLM(
		mockResponse{Err: fmt.Errorf("llm failure")},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"failer","task":"fail"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "err-hooks"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"failer": agent.New(failLLM,
				agent.WithSystemPrompt("Failer"),
			),
		}),
		agent.WithHooks(agent.Hooks{
			OnTeammateError: func(
				_ context.Context,
				tc agent.TeammateEventContext,
			) {
				mu.Lock()
				errorFired = true
				if tc.Error != nil {
					capturedErr = tc.Error.Error()
				}
				mu.Unlock()
			},
		}),
	)

	_, err := lead.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if !errorFired {
		t.Error("expected OnTeammateError hook to fire")
	}
	if capturedErr == "" {
		t.Error("expected error message in context")
	}
}

func TestTeamHooks_ObservingHooks(t *testing.T) {
	var mu sync.Mutex
	events := map[string]bool{}

	teammateLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-t1",
					Name:  "send_message",
					Input: `{"to":"__lead__","content":"hi"}`,
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
					Input: `{"name":"obs","task":"observe"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "obs-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"obs": agent.New(teammateLLM,
				agent.WithSystemPrompt("Observer"),
			),
		}),
		agent.WithHooks(
			agent.NewObservingHooks(func(evt agent.HookEvent) {
				mu.Lock()
				events[string(evt.Type)] = true
				mu.Unlock()
			}),
		),
	)

	_, err := lead.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	for _, evtType := range []string{
		"teammate_join",
		"teammate_complete",
		"team_message",
	} {
		if !events[evtType] {
			t.Errorf(
				"expected observing hook event %q",
				evtType,
			)
		}
	}
}

func TestTeamHooks_ObservingHooksError(t *testing.T) {
	var mu sync.Mutex
	events := map[string]bool{}

	failLLM := newMockLLM(
		mockResponse{Err: fmt.Errorf("llm failure")},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"failer","task":"fail"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "done"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "obs-err-team"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"failer": agent.New(failLLM,
				agent.WithSystemPrompt("Failer"),
			),
		}),
		agent.WithHooks(
			agent.NewObservingHooks(func(evt agent.HookEvent) {
				mu.Lock()
				events[string(evt.Type)] = true
				mu.Unlock()
			}),
		),
	)

	_, err := lead.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if !events["teammate_error"] {
		t.Error("expected teammate_error observing event")
	}
}

func TestTeamRunner_StreamErrorEvent(t *testing.T) {
	failLLM := newMockLLM(
		mockResponse{Err: fmt.Errorf("llm failure")},
	)

	leadLLM := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc-1",
					Name:  "spawn_teammate",
					Input: `{"name":"failer","task":"fail"}`,
					Type:  "function",
				},
			},
		},
		mockResponse{Content: "handled"},
	)

	lead := agent.New(leadLLM,
		agent.WithSystemPrompt("Lead"),
		agent.WithTeam(team.Config{Name: "stream-err"}),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"failer": agent.New(failLLM,
				agent.WithSystemPrompt("Failer"),
			),
		}),
	)

	var sawError bool
	for event := range lead.ChatStream(
		context.Background(),
		"go",
	) {
		if event.Type == types.EventTeammateError {
			sawError = true
		}
	}

	if !sawError {
		t.Error("expected EventTeammateError stream event")
	}
}
