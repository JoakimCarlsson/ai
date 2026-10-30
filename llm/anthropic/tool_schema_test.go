package anthropic

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/tool"
)

// requiredTool is a BaseTool whose schema names one required parameter and
// one optional one.
type requiredTool struct{ required []string }

// Info describes a weather lookup with a required location and an optional
// unit.
func (r requiredTool) Info() tool.Info {
	return tool.Info{
		Name:        "get_weather",
		Description: "d",
		Parameters: map[string]any{
			"location": map[string]any{"type": "string"},
			"unit":     map[string]any{"type": "string"},
		},
		Required: r.required,
	}
}

// Run does nothing; the tests only marshal the tool's schema.
func (r requiredTool) Run(context.Context, tool.Call) (tool.Response, error) {
	return tool.Response{}, nil
}

// inputSchema returns the input_schema of the first tool in the request body.
func inputSchema(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	tools, ok := body["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools = %v, want one tool", body["tools"])
	}
	first, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("tools[0] = %v, want an object", tools[0])
	}
	schema, ok := first["input_schema"].(map[string]any)
	if !ok {
		t.Fatalf("input_schema = %v, want an object", first["input_schema"])
	}
	return schema
}

// TestToolSchemaCarriesRequired confirms a tool's required parameters reach
// the request's input_schema, so the model is told which ones it must fill.
func TestToolSchemaCarriesRequired(t *testing.T) {
	body := toolChoiceBody(t,
		[]tool.BaseTool{requiredTool{required: []string{"location"}}})

	got, ok := inputSchema(t, body)["required"].([]any)
	if !ok || len(got) != 1 || got[0] != "location" {
		t.Fatalf("required = %v, want [location]",
			inputSchema(t, body)["required"])
	}
}

// TestToolSchemaWithNoRequiredOmitsTheKey confirms a tool with no required
// parameters sends no required key rather than an empty list.
func TestToolSchemaWithNoRequiredOmitsTheKey(t *testing.T) {
	body := toolChoiceBody(t, []tool.BaseTool{requiredTool{}})

	if got, present := inputSchema(t, body)["required"]; present {
		t.Fatalf("required = %v, want the key omitted", got)
	}
}
