package tool

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/tool"
)

func TestNewToolset_ReturnsAllTools(t *testing.T) {
	ts := tool.NewToolset("basics",
		&stubTool{name: "a", output: "1"},
		&stubTool{name: "b", output: "2"},
	)

	if ts.Name() != "basics" {
		t.Errorf("expected name 'basics', got %q", ts.Name())
	}

	tools := ts.Tools(context.Background())
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}

func TestFilterToolset_FiltersTools(t *testing.T) {
	inner := tool.NewToolset("all",
		&stubTool{name: "allowed", output: "ok"},
		&stubTool{name: "blocked", output: "no"},
	)

	filtered := tool.NewFilterToolset("filtered", inner,
		func(_ context.Context, t tool.BaseTool) bool {
			return t.Info().Name == "allowed"
		},
	)

	if filtered.Name() != "filtered" {
		t.Errorf("expected name 'filtered', got %q", filtered.Name())
	}

	tools := filtered.Tools(context.Background())
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Info().Name != "allowed" {
		t.Errorf("expected 'allowed', got %q", tools[0].Info().Name)
	}
}

type contextKey string

func TestFilterToolset_PredicateReceivesContext(t *testing.T) {
	inner := tool.NewToolset("tools",
		&stubTool{name: "recon", output: ""},
		&stubTool{name: "exploit", output: ""},
	)

	key := contextKey("phase")
	filtered := tool.NewFilterToolset("phase-aware", inner,
		func(ctx context.Context, bt tool.BaseTool) bool {
			phase, _ := ctx.Value(key).(string)
			if bt.Info().Name == "exploit" {
				return phase == "exploitation"
			}
			return true
		},
	)

	reconCtx := context.WithValue(context.Background(), key, "recon")
	tools := filtered.Tools(reconCtx)
	if len(tools) != 1 {
		t.Fatalf("recon phase: expected 1 tool, got %d", len(tools))
	}
	if tools[0].Info().Name != "recon" {
		t.Errorf("expected 'recon', got %q", tools[0].Info().Name)
	}

	exploitCtx := context.WithValue(context.Background(), key, "exploitation")
	tools = filtered.Tools(exploitCtx)
	if len(tools) != 2 {
		t.Fatalf("exploitation phase: expected 2 tools, got %d", len(tools))
	}
}

func TestFilterToolset_EmptyWhenAllFiltered(t *testing.T) {
	inner := tool.NewToolset("tools",
		&stubTool{name: "a", output: ""},
	)

	filtered := tool.NewFilterToolset("none", inner,
		func(_ context.Context, _ tool.BaseTool) bool { return false },
	)

	tools := filtered.Tools(context.Background())
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestCompositeToolset_MergesChildren(t *testing.T) {
	ts1 := tool.NewToolset("set1", &stubTool{name: "a", output: ""})
	ts2 := tool.NewToolset("set2", &stubTool{name: "b", output: ""})

	composite := tool.NewCompositeToolset("combined", ts1, ts2)

	if composite.Name() != "combined" {
		t.Errorf("expected name 'combined', got %q", composite.Name())
	}

	tools := composite.Tools(context.Background())
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tt := range tools {
		names[tt.Info().Name] = true
	}
	if !names["a"] || !names["b"] {
		t.Errorf("expected tools a and b, got %v", names)
	}
}

func TestCompositeToolset_WithFilteredChild(t *testing.T) {
	inner := tool.NewToolset("tools",
		&stubTool{name: "keep", output: ""},
		&stubTool{name: "drop", output: ""},
	)
	filtered := tool.NewFilterToolset("filtered", inner,
		func(_ context.Context, bt tool.BaseTool) bool {
			return bt.Info().Name == "keep"
		},
	)
	extra := tool.NewToolset("extra", &stubTool{name: "bonus", output: ""})

	composite := tool.NewCompositeToolset("mixed", filtered, extra)
	tools := composite.Tools(context.Background())
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}

func TestNewToolset_Empty(t *testing.T) {
	ts := tool.NewToolset("empty")
	tools := ts.Tools(context.Background())
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}
