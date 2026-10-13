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

const orDetails = `[{"type":"reasoning.text","text":"a","signature":"sig","format":"anthropic-claude-v1","index":0},` +
	`{"type":"reasoning.encrypted","data":"blob","format":"anthropic-claude-v1","index":1}]`

// TestChatRoundTripsReasoningDetails verifies the complete ordered array is
// stored, survives JSON, and is replayed unchanged after a tool result.
func TestChatRoundTripsReasoningDetails(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			bodies = append(bodies, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"",`+
				`"reasoning":"a","reasoning_details":`+orDetails+`,`+
				`"tool_calls":[{"id":"call_1","type":"function","function":{"name":"f","arguments":"{}"}}]},`+
				`"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
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
	assistant := message.NewAssistantMessage()
	for _, item := range resp.ReasoningItems {
		assistant.Parts = append(assistant.Parts, item)
	}
	assistant.AppendToolCalls(resp.ToolCalls)
	data, _ := json.Marshal(assistant)
	var stored message.Message
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	result := message.NewMessage(message.Tool, nil)
	result.AddToolResult(message.ToolResult{ToolCallID: "call_1", Content: "ok"})
	if _, err := client.SendMessages(
		context.Background(),
		[]message.Message{history[0], stored, result},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	var want any
	_ = json.Unmarshal([]byte(orDetails), &want)
	for _, m := range bodies[1]["messages"].([]any) {
		wire := m.(map[string]any)
		if wire["role"] != "assistant" {
			continue
		}
		if !reflect.DeepEqual(wire["reasoning_details"], want) {
			t.Errorf("reasoning_details = %v", wire["reasoning_details"])
		}
		if _, ok := wire["reasoning"]; ok {
			t.Errorf("plain reasoning sent alongside details: %v", wire)
		}
		return
	}
	t.Fatal("no assistant message")
}

// TestNewResponsesLLM verifies the endpoint, stateless request and replay.
func TestNewResponsesLLM(t *testing.T) {
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
	client := NewResponsesLLM(
		openai.WithResponsesAPIKey("k"),
		openai.WithResponsesBaseURL(srv.URL),
		openai.WithResponsesModel(llm.Model{APIModel: "m", CanReason: true}),
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
