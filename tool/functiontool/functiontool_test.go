package functiontool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/joakimcarlsson/ai/tool"
)

type greetParams struct {
	Name string `json:"name" desc:"Person to greet"`
}

func TestNew_ContextAndParams_StringReturn(t *testing.T) {
	ft := New(
		"greet",
		"Greet someone",
		func(_ context.Context, p greetParams) (string, error) {
			return "Hello " + p.Name, nil
		},
	)

	info := ft.Info()
	if info.Name != "greet" {
		t.Errorf("expected name 'greet', got %q", info.Name)
	}
	if info.Description != "Greet someone" {
		t.Errorf(
			"expected description 'Greet someone', got %q",
			info.Description,
		)
	}
	if info.Parameters == nil {
		t.Fatal("expected non-nil parameters")
	}
	if _, ok := info.Parameters["name"]; !ok {
		t.Error("expected 'name' in parameters")
	}
	if len(info.Required) != 1 || info.Required[0] != "name" {
		t.Errorf("expected required=[name], got %v", info.Required)
	}

	resp, err := ft.Run(
		context.Background(),
		tool.Call{ID: "1", Name: "greet", Input: `{"name":"Alice"}`},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello Alice" {
		t.Errorf("expected 'Hello Alice', got %q", resp.Content)
	}
	if resp.IsError {
		t.Error("expected IsError=false")
	}
}

func TestNew_ParamsOnly_StringReturn(t *testing.T) {
	ft := New("greet", "Greet", func(p greetParams) (string, error) {
		return "Hi " + p.Name, nil
	})

	resp, err := ft.Run(
		context.Background(),
		tool.Call{Input: `{"name":"Bob"}`},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hi Bob" {
		t.Errorf("expected 'Hi Bob', got %q", resp.Content)
	}
}

func TestNew_ContextOnly_NoParams(t *testing.T) {
	ft := New("ping", "Ping", func(_ context.Context) (string, error) {
		return "pong", nil
	})

	info := ft.Info()
	if info.Parameters != nil {
		t.Errorf("expected nil parameters, got %v", info.Parameters)
	}

	resp, err := ft.Run(context.Background(), tool.Call{Input: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "pong" {
		t.Errorf("expected 'pong', got %q", resp.Content)
	}
}

func TestNew_NoInputs(t *testing.T) {
	ft := New("hello", "Hello", func() (string, error) {
		return "world", nil
	})

	info := ft.Info()
	if info.Parameters != nil {
		t.Errorf("expected nil parameters, got %v", info.Parameters)
	}

	resp, err := ft.Run(context.Background(), tool.Call{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "world" {
		t.Errorf("expected 'world', got %q", resp.Content)
	}
}

func TestNew_ResponseReturn(t *testing.T) {
	ft := New("img", "Return image", func() (tool.Response, error) {
		return tool.NewImageResponse("base64data"), nil
	})

	resp, err := ft.Run(context.Background(), tool.Call{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Type != tool.ResponseTypeImage {
		t.Errorf("expected image type, got %s", resp.Type)
	}
	if resp.Content != "base64data" {
		t.Errorf("expected 'base64data', got %q", resp.Content)
	}
}

type userResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestNew_JSONReturn(t *testing.T) {
	ft := New("user", "Get user", func(p greetParams) (userResult, error) {
		return userResult{ID: 1, Name: p.Name}, nil
	})

	resp, err := ft.Run(
		context.Background(),
		tool.Call{Input: `{"name":"Eve"}`},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Type != tool.ResponseTypeJSON {
		t.Errorf("expected json type, got %s", resp.Type)
	}

	var result userResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Name != "Eve" || result.ID != 1 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestNew_ErrorReturn(t *testing.T) {
	ft := New("fail", "Fail", func() (string, error) {
		return "", errors.New("something broke")
	})

	resp, err := ft.Run(context.Background(), tool.Call{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !resp.IsError {
		t.Error("expected IsError=true")
	}
	if resp.Content != "something broke" {
		t.Errorf("expected 'something broke', got %q", resp.Content)
	}
}

func TestNew_ConfirmationRejected_PropagatedAsGoError(t *testing.T) {
	ft := New("dangerous", "Dangerous", func() (string, error) {
		return "", tool.ErrConfirmationRejected
	})

	_, err := ft.Run(context.Background(), tool.Call{})
	if !errors.Is(err, tool.ErrConfirmationRejected) {
		t.Errorf("expected ErrConfirmationRejected, got %v", err)
	}
}

func TestNew_InvalidJSON(t *testing.T) {
	ft := New("greet", "Greet", func(p greetParams) (string, error) {
		return "Hi " + p.Name, nil
	})

	resp, err := ft.Run(context.Background(), tool.Call{Input: `{invalid`})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !resp.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestNew_PanicRecovery(t *testing.T) {
	ft := New("boom", "Boom", func() (string, error) {
		panic("kaboom")
	})

	resp, err := ft.Run(context.Background(), tool.Call{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !resp.IsError {
		t.Error("expected IsError=true for panic")
	}
	if resp.Content != "tool panicked: kaboom" {
		t.Errorf("expected panic message, got %q", resp.Content)
	}
}

func TestWithConfirmation(t *testing.T) {
	ft := New("del", "Delete", func() (string, error) {
		return "deleted", nil
	}, WithConfirmation())

	if !ft.Info().RequireConfirmation {
		t.Error("expected RequireConfirmation=true")
	}
}

func TestNew_PanicsOnNonFunction(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-function")
		}
	}()
	New("bad", "Bad", "not a function")
}

func TestNew_PanicsOnTooManyInputs(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for too many inputs")
		}
	}()
	New(
		"bad",
		"Bad",
		func(_ context.Context, _ greetParams, _ string) (string, error) {
			return "", nil
		},
	)
}

func TestNew_PanicsOnWrongReturnCount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for wrong return count")
		}
	}()
	New("bad", "Bad", func() string { return "" })
}

func TestNew_PanicsOnNonErrorSecondReturn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-error second return")
		}
	}()
	New("bad", "Bad", func() (string, string) { return "", "" })
}

func TestNew_PanicsOnNonStructParam(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-struct param")
		}
	}()
	New("bad", "Bad", func(s string) (string, error) { return s, nil })
}

func TestNew_RegistryIntegration(t *testing.T) {
	ft := New("echo", "Echo", func(p greetParams) (string, error) {
		return p.Name, nil
	})

	reg := tool.NewRegistry()
	reg.Register(ft)

	resp, err := reg.Execute(context.Background(), tool.Call{
		ID:    "1",
		Name:  "echo",
		Input: `{"name":"test"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "test" {
		t.Errorf("expected 'test', got %q", resp.Content)
	}
}
