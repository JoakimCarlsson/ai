package gemini

import (
	"testing"

	"github.com/joakimcarlsson/ai/model"
)

func reasoningClient(opts ...Option) *Client {
	o := Options{model: model.Model{CanReason: true}}
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{options: o}
}

// TestThinkingBudgetSetsConfig verifies WithThinkingBudget populates
// ThinkingConfig.ThinkingBudget on the built config.
func TestThinkingBudgetSetsConfig(t *testing.T) {
	cfg := reasoningClient(WithThinkingBudget(2048)).buildConfig(nil, nil)
	if cfg.ThinkingConfig == nil {
		t.Fatal("expected ThinkingConfig to be set")
	}
	if cfg.ThinkingConfig.ThinkingBudget == nil {
		t.Fatal("expected ThinkingBudget to be set")
	}
	if got := *cfg.ThinkingConfig.ThinkingBudget; got != 2048 {
		t.Errorf("ThinkingBudget = %d, want 2048", got)
	}
}

// TestThinkingBudgetZeroDisables verifies a budget of 0 is sent as an explicit
// 0 (disable thinking), not omitted.
func TestThinkingBudgetZeroDisables(t *testing.T) {
	cfg := reasoningClient(WithThinkingBudget(0)).buildConfig(nil, nil)
	if cfg.ThinkingConfig == nil || cfg.ThinkingConfig.ThinkingBudget == nil {
		t.Fatal("expected ThinkingBudget to be set to 0")
	}
	if got := *cfg.ThinkingConfig.ThinkingBudget; got != 0 {
		t.Errorf("ThinkingBudget = %d, want 0", got)
	}
}

// TestThinkingLevelStillWorks verifies the named-level option keeps mapping to
// the SDK ThinkingLevel and combines with a budget.
func TestThinkingLevelStillWorks(t *testing.T) {
	cfg := reasoningClient(WithThinkingLevel(ThinkingLevelHigh)).
		buildConfig(nil, nil)
	if cfg.ThinkingConfig == nil {
		t.Fatal("expected ThinkingConfig to be set")
	}
	if cfg.ThinkingConfig.ThinkingLevel == "" {
		t.Error("expected ThinkingLevel to be set")
	}
	if cfg.ThinkingConfig.ThinkingBudget != nil {
		t.Error("expected ThinkingBudget to remain unset")
	}
}

// TestThinkingDisabledWithoutReasoning verifies no thinking config leaks onto a
// model that cannot reason.
func TestThinkingDisabledWithoutReasoning(t *testing.T) {
	c := &Client{options: Options{model: model.Model{CanReason: false}}}
	WithThinkingBudget(1024)(&c.options)
	cfg := c.buildConfig(nil, nil)
	if cfg.ThinkingConfig != nil {
		t.Error("expected no ThinkingConfig when model cannot reason")
	}
}
