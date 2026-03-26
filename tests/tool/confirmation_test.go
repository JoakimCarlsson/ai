package tool

import (
	"context"
	"errors"
	"testing"

	"github.com/joakimcarlsson/ai/tool"
)

func TestRequestConfirmation_NoHandler(t *testing.T) {
	err := tool.RequestConfirmation(
		context.Background(),
		"delete everything",
		nil,
	)
	if err != nil {
		t.Fatalf("expected nil (auto-approve) when no handler, got %v", err)
	}
}

func TestRequestConfirmation_Approved(t *testing.T) {
	var capturedHint string
	var capturedPayload any

	handler := func(hint string, payload any) error {
		capturedHint = hint
		capturedPayload = payload
		return nil
	}
	ctx := tool.WithConfirmationHandler(context.Background(), handler)

	payload := map[string]string{"table": "users"}
	err := tool.RequestConfirmation(ctx, "delete records", payload)
	if err != nil {
		t.Fatalf("expected nil for approved confirmation, got %v", err)
	}
	if capturedHint != "delete records" {
		t.Errorf("expected hint 'delete records', got %q", capturedHint)
	}
	if capturedPayload == nil {
		t.Error("expected payload to be passed through")
	}
}

func TestRequestConfirmation_Rejected(t *testing.T) {
	handler := func(string, any) error {
		return tool.ErrConfirmationRejected
	}
	ctx := tool.WithConfirmationHandler(context.Background(), handler)

	err := tool.RequestConfirmation(ctx, "dangerous op", nil)
	if !errors.Is(err, tool.ErrConfirmationRejected) {
		t.Fatalf("expected ErrConfirmationRejected, got %v", err)
	}
}

func TestWithConfirmationToolset_SetsFlag(t *testing.T) {
	inner := tool.NewToolset("ops",
		&stubTool{name: "safe", output: "ok"},
		&stubTool{name: "dangerous", output: "boom"},
	)

	wrapped := tool.WithConfirmation(inner)

	if wrapped.Name() != "ops" {
		t.Errorf("expected name 'ops', got %q", wrapped.Name())
	}

	tools := wrapped.Tools(context.Background())
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	for _, tt := range tools {
		if !tt.Info().RequireConfirmation {
			t.Errorf(
				"tool %q should have RequireConfirmation=true",
				tt.Info().Name,
			)
		}
	}
}

func TestWithConfirmationToolset_DelegatesRun(t *testing.T) {
	inner := tool.NewToolset("ops",
		&stubTool{name: "echo", output: "hello"},
	)

	wrapped := tool.WithConfirmation(inner)
	tools := wrapped.Tools(context.Background())

	resp, err := tools[0].Run(context.Background(), tool.Call{
		ID:    "1",
		Name:  "echo",
		Input: "{}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("expected 'hello', got %q", resp.Content)
	}
}

func TestWithConfirmationToolset_OriginalUnchanged(t *testing.T) {
	inner := tool.NewToolset("ops",
		&stubTool{name: "a", output: ""},
	)

	_ = tool.WithConfirmation(inner)

	tools := inner.Tools(context.Background())
	if tools[0].Info().RequireConfirmation {
		t.Error("original tool should not have RequireConfirmation set")
	}
}
