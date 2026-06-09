package gemini

import (
	"context"
	"errors"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
	"google.golang.org/genai"
)

// stubTool is a no-op BaseTool used to populate the request's tools slice so
// tool-choice assertions have something to attach to.
type stubTool struct{ name string }

func (s stubTool) Info() tool.Info {
	return tool.Info{
		Name:        s.name,
		Description: "d",
		Parameters:  map[string]any{},
	}
}

func (s stubTool) Run(context.Context, tool.Call) (tool.Response, error) {
	return tool.Response{}, nil
}

func clientWith(opts ...Option) *Client {
	o := Options{}
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{options: o}
}

func functionCallingConfig(
	t *testing.T,
	cfg *genai.GenerateContentConfig,
) *genai.FunctionCallingConfig {
	t.Helper()
	if cfg.ToolConfig == nil || cfg.ToolConfig.FunctionCallingConfig == nil {
		t.Fatal("expected toolConfig.functionCallingConfig to be set")
	}
	return cfg.ToolConfig.FunctionCallingConfig
}

// TestToolChoiceRequired confirms a Required choice maps mode to ANY.
func TestToolChoiceRequired(t *testing.T) {
	cfg := clientWith(
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
	).buildConfig(nil, []tool.BaseTool{stubTool{name: "get_weather"}})

	fc := functionCallingConfig(t, cfg)
	if fc.Mode != genai.FunctionCallingConfigModeAny {
		t.Errorf("mode = %q, want ANY", fc.Mode)
	}
	if len(fc.AllowedFunctionNames) != 0 {
		t.Errorf(
			"AllowedFunctionNames = %v, want empty",
			fc.AllowedFunctionNames,
		)
	}
}

// TestToolChoiceNone confirms a None choice maps mode to NONE.
func TestToolChoiceNone(t *testing.T) {
	cfg := clientWith(
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceNone}),
	).buildConfig(nil, []tool.BaseTool{stubTool{name: "get_weather"}})

	if fc := functionCallingConfig(
		t,
		cfg,
	); fc.Mode != genai.FunctionCallingConfigModeNone {
		t.Errorf("mode = %q, want NONE", fc.Mode)
	}
}

// TestToolChoiceSpecific confirms a Specific choice maps mode to ANY and lists
// the named tool in allowedFunctionNames.
func TestToolChoiceSpecific(t *testing.T) {
	cfg := clientWith(
		WithToolChoice(llm.ToolChoice{
			Mode: llm.ToolChoiceSpecific,
			Name: "get_weather",
		}),
	).buildConfig(nil, []tool.BaseTool{stubTool{name: "get_weather"}})

	fc := functionCallingConfig(t, cfg)
	if fc.Mode != genai.FunctionCallingConfigModeAny {
		t.Errorf("mode = %q, want ANY", fc.Mode)
	}
	if len(fc.AllowedFunctionNames) != 1 ||
		fc.AllowedFunctionNames[0] != "get_weather" {
		t.Errorf("AllowedFunctionNames = %v, want [get_weather]",
			fc.AllowedFunctionNames)
	}
}

// TestToolChoiceOmittedWithoutTools confirms no toolConfig is emitted when the
// tools slice is empty.
func TestToolChoiceOmittedWithoutTools(t *testing.T) {
	cfg := clientWith(
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
	).buildConfig(nil, nil)

	if cfg.ToolConfig != nil {
		t.Errorf("toolConfig should be omitted with no tools, got %v",
			cfg.ToolConfig)
	}
}

// TestToolChoiceSpecificEmptyNameRejected confirms a Specific choice with no
// name is rejected before any request is sent.
func TestToolChoiceSpecificEmptyNameRejected(t *testing.T) {
	c := clientWith(
		WithModel(model.Model{APIModel: "gemini-2.5-flash"}),
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceSpecific}),
	)

	_, err := c.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")},
		[]tool.BaseTool{stubTool{name: "get_weather"}})
	if !errors.Is(err, llm.ErrToolChoiceNameRequired) {
		t.Fatalf("expected ErrToolChoiceNameRequired, got %v", err)
	}
}
