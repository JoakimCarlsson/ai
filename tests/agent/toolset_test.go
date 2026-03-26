package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

func collectToolNames(llmClient *toolCapturingLLM) *[][]string {
	var mu sync.Mutex
	var calls [][]string
	llmClient.onTools = func(names []string) {
		mu.Lock()
		cp := make([]string, len(names))
		copy(cp, names)
		calls = append(calls, cp)
		mu.Unlock()
	}
	return &calls
}

type phaseKey struct{}

func TestToolset_StaticToolsetProvided(t *testing.T) {
	ts := tool.NewToolset("basics", &echoTool{})

	base := newMockLLM(mockResponse{Content: "done"})
	llmClient := &toolCapturingLLM{base: base}
	calls := collectToolNames(llmClient)

	a := agent.New(llmClient, agent.WithToolsets(ts))

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*calls) == 0 {
		t.Fatal("expected at least one LLM call")
	}
	names := (*calls)[0]
	if len(names) != 1 || names[0] != "echo" {
		t.Fatalf("expected [echo], got %v", names)
	}
}

func TestToolset_MixedToolsAndToolsets(t *testing.T) {
	ts := tool.NewToolset("set", &echoTool{})
	extra := &simpleTool{
		name: "ping",
		run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
			return tool.NewTextResponse("pong"), nil
		},
	}

	base := newMockLLM(mockResponse{Content: "done"})
	llmClient := &toolCapturingLLM{base: base}
	calls := collectToolNames(llmClient)

	a := agent.New(llmClient,
		agent.WithTools(extra),
		agent.WithToolsets(ts),
	)

	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := (*calls)[0]
	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["ping"] || !nameSet["echo"] {
		t.Fatalf("expected both ping and echo, got %v", names)
	}
}

func TestToolset_FilterByContext(t *testing.T) {
	inner := tool.NewToolset("pentest",
		&simpleTool{
			name: "scan",
			run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
				return tool.NewTextResponse("scanned"), nil
			},
		},
		&simpleTool{
			name: "exploit",
			run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
				return tool.NewTextResponse("exploited"), nil
			},
		},
	)

	filtered := tool.NewFilterToolset("phase-aware", inner,
		func(ctx context.Context, bt tool.BaseTool) bool {
			phase, _ := ctx.Value(phaseKey{}).(string)
			if bt.Info().Name == "exploit" {
				return phase == "exploitation"
			}
			return true
		},
	)

	t.Run("recon phase only scan available", func(t *testing.T) {
		base := newMockLLM(mockResponse{Content: "done"})
		llmClient := &toolCapturingLLM{base: base}
		calls := collectToolNames(llmClient)

		a := agent.New(llmClient, agent.WithToolsets(filtered))

		ctx := context.WithValue(context.Background(), phaseKey{}, "recon")
		_, err := a.Chat(ctx, "scan target")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		names := (*calls)[0]
		if len(names) != 1 || names[0] != "scan" {
			t.Fatalf("recon phase: expected [scan], got %v", names)
		}
	})

	t.Run("exploitation phase both available", func(t *testing.T) {
		base := newMockLLM(mockResponse{Content: "done"})
		llmClient := &toolCapturingLLM{base: base}
		calls := collectToolNames(llmClient)

		a := agent.New(llmClient, agent.WithToolsets(filtered))

		ctx := context.WithValue(context.Background(), phaseKey{}, "exploitation")
		_, err := a.Chat(ctx, "exploit target")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		names := (*calls)[0]
		nameSet := map[string]bool{}
		for _, n := range names {
			nameSet[n] = true
		}
		if !nameSet["scan"] || !nameSet["exploit"] {
			t.Fatalf("exploitation phase: expected scan and exploit, got %v", names)
		}
	})
}

func TestToolset_FilterChangesPerTurn(t *testing.T) {
	var mu sync.Mutex
	phase := "recon"

	inner := tool.NewToolset("tools",
		&simpleTool{
			name: "scan",
			run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
				mu.Lock()
				phase = "exploitation"
				mu.Unlock()
				return tool.NewTextResponse("scanned"), nil
			},
		},
		&simpleTool{
			name: "exploit",
			run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
				return tool.NewTextResponse("exploited"), nil
			},
		},
	)

	filtered := tool.NewFilterToolset("dynamic", inner,
		func(_ context.Context, bt tool.BaseTool) bool {
			mu.Lock()
			p := phase
			mu.Unlock()
			if bt.Info().Name == "exploit" {
				return p == "exploitation"
			}
			return true
		},
	)

	base := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "scan", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)
	llmClient := &toolCapturingLLM{base: base}
	calls := collectToolNames(llmClient)

	a := agent.New(llmClient, agent.WithToolsets(filtered))

	_, err := a.Chat(context.Background(), "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	turn1Names := (*calls)[0]
	if len(turn1Names) != 1 || turn1Names[0] != "scan" {
		t.Fatalf("turn 1: expected [scan], got %v", turn1Names)
	}

	turn2Names := (*calls)[1]
	nameSet := map[string]bool{}
	for _, n := range turn2Names {
		nameSet[n] = true
	}
	if !nameSet["scan"] || !nameSet["exploit"] {
		t.Fatalf("turn 2: expected scan and exploit, got %v", turn2Names)
	}
}

func TestToolset_FilterChangesBetweenCalls(t *testing.T) {
	inner := tool.NewToolset("tools",
		&simpleTool{
			name: "scan",
			run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
				return tool.NewTextResponse("scanned"), nil
			},
		},
		&simpleTool{
			name: "exploit",
			run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
				return tool.NewTextResponse("exploited"), nil
			},
		},
	)

	filtered := tool.NewFilterToolset("dynamic", inner,
		func(ctx context.Context, bt tool.BaseTool) bool {
			phase, _ := ctx.Value(phaseKey{}).(string)
			if bt.Info().Name == "exploit" {
				return phase == "exploitation"
			}
			return true
		},
	)

	base := newMockLLM(
		mockResponse{Content: "recon done"},
		mockResponse{Content: "exploit done"},
	)
	llmClient := &toolCapturingLLM{base: base}
	calls := collectToolNames(llmClient)

	a := agent.New(llmClient, agent.WithToolsets(filtered))

	reconCtx := context.WithValue(context.Background(), phaseKey{}, "recon")
	_, err := a.Chat(reconCtx, "scan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call1Names := (*calls)[0]
	if len(call1Names) != 1 || call1Names[0] != "scan" {
		t.Fatalf("call 1: expected [scan], got %v", call1Names)
	}

	exploitCtx := context.WithValue(context.Background(), phaseKey{}, "exploitation")
	_, err = a.Chat(exploitCtx, "exploit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call2Names := (*calls)[1]
	nameSet := map[string]bool{}
	for _, n := range call2Names {
		nameSet[n] = true
	}
	if !nameSet["scan"] || !nameSet["exploit"] {
		t.Fatalf("call 2: expected scan and exploit, got %v", call2Names)
	}
}

func TestToolset_CompositeInAgent(t *testing.T) {
	ts1 := tool.NewToolset("recon", &simpleTool{
		name: "scan",
		run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
			return tool.NewTextResponse("scanned"), nil
		},
	})
	ts2 := tool.NewToolset("exploit", &simpleTool{
		name: "inject",
		run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
			return tool.NewTextResponse("injected"), nil
		},
	})

	composite := tool.NewCompositeToolset("suite", ts1, ts2)

	base := newMockLLM(mockResponse{Content: "done"})
	llmClient := &toolCapturingLLM{base: base}
	calls := collectToolNames(llmClient)

	a := agent.New(llmClient, agent.WithToolsets(composite))

	_, err := a.Chat(context.Background(), "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := (*calls)[0]
	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["scan"] || !nameSet["inject"] {
		t.Fatalf("expected scan and inject, got %v", names)
	}
}

func TestToolset_ToolExecutionFromToolset(t *testing.T) {
	var executed bool
	ts := tool.NewToolset("tools", &simpleTool{
		name: "action",
		run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
			executed = true
			return tool.NewTextResponse("ran"), nil
		},
	})

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "action", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock, agent.WithToolsets(ts))

	resp, err := a.Chat(context.Background(), "do it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executed {
		t.Fatal("tool from toolset should have been executed")
	}
	if resp.TotalToolCalls != 1 {
		t.Fatalf("expected 1 tool call, got %d", resp.TotalToolCalls)
	}
}

func TestToolset_HooksApplyToToolsetTools(t *testing.T) {
	var denied bool
	ts := tool.NewToolset("tools", &simpleTool{
		name: "dangerous",
		run: func(_ context.Context, _ tool.Call) (tool.Response, error) {
			return tool.NewTextResponse("should not run"), nil
		},
	})

	hooks := agent.Hooks{
		PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
			if tc.ToolName == "dangerous" {
				denied = true
				return agent.PreToolUseResult{
					Action:     agent.HookDeny,
					DenyReason: "blocked by policy",
				}, nil
			}
			return agent.PreToolUseResult{Action: agent.HookAllow}, nil
		},
	}

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "dangerous", Input: `{}`, Type: "function"},
			},
		},
		mockResponse{Content: "ok"},
	)

	a := agent.New(mock,
		agent.WithToolsets(ts),
		agent.WithHooks(hooks),
	)

	_, err := a.Chat(context.Background(), "try it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !denied {
		t.Fatal("hook should have denied tool from toolset")
	}
}

func TestToolset_StreamingWithToolset(t *testing.T) {
	ts := tool.NewToolset("tools", &echoTool{})

	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc-1", Name: "echo", Input: `{"text":"stream"}`, Type: "function"},
			},
		},
		mockResponse{Content: "streamed"},
	)

	a := agent.New(mock, agent.WithToolsets(ts))

	var finalContent string
	for event := range a.ChatStream(context.Background(), "test") {
		if event.Type == types.EventComplete && event.Response != nil {
			finalContent = event.Response.Content
		}
	}

	if finalContent != "streamed" {
		t.Fatalf("expected 'streamed', got %q", finalContent)
	}
}

func TestToolset_EmptyToolset(t *testing.T) {
	ts := tool.NewToolset("empty")

	base := newMockLLM(mockResponse{Content: "no tools"})
	llmClient := &toolCapturingLLM{base: base}
	calls := collectToolNames(llmClient)

	a := agent.New(llmClient, agent.WithToolsets(ts))

	resp, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "no tools" {
		t.Fatalf("expected 'no tools', got %q", resp.Content)
	}

	names := (*calls)[0]
	if len(names) != 0 {
		t.Fatalf("expected 0 tools, got %v", names)
	}
}
