package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
)

// TestNewLLM verifies the endpoint, stateless request and replay.
func TestNewLLM(t *testing.T) {
	var path string
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			path = r.URL.Path
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			bodies = append(bodies, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"r","object":"response","status":"completed","output":[`+
				`{"type":"reasoning","id":"rs_1","encrypted_content":"enc","summary":[]},`+
				`{"type":"function_call","id":"fc","call_id":"call_1","name":"f","arguments":"{}","status":"completed"}],`+
				`"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`)
		}))
	defer srv.Close()
	client := NewLLM(
		openai.WithAPIKey("k"),
		openai.WithBaseURL(srv.URL),
		openai.WithModel(llm.Model{APIModel: "m", CanReason: true}),
	)
	history := []message.Message{message.NewUserMessage("go")}
	resp, err := client.SendMessages(context.Background(), history, nil)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/responses" || bodies[0]["store"] != false ||
		!reflect.DeepEqual(bodies[0]["include"], []any{"reasoning.encrypted_content"}) {
		t.Errorf("path = %s body = %v", path, bodies[0])
	}
	if _, ok := bodies[0]["previous_response_id"]; ok {
		t.Error("previous_response_id sent")
	}
	assistant := message.NewAssistantMessage()
	for _, item := range resp.ReasoningItems {
		assistant.Parts = append(assistant.Parts, item)
	}
	assistant.AppendToolCalls(resp.ToolCalls)
	result := message.NewMessage(message.Tool, nil)
	result.AddToolResult(message.ToolResult{ToolCallID: "call_1", Content: "ok"})
	if _, err := client.SendMessages(
		context.Background(),
		[]message.Message{history[0], assistant, result},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	input := bodies[1]["input"].([]any)
	if input[1].(map[string]any)["encrypted_content"] != "enc" {
		t.Errorf("input = %v", input)
	}
}
