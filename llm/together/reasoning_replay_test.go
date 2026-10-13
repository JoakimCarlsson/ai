package together

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
)

const replayReply = `{"choices":[{"index":0,"message":{"role":"assistant","content":"","reasoning":"think",` +
	`"tool_calls":[{"id":"call_1","type":"function","function":{"name":"f","arguments":"{}"}}]},` +
	`"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`

// replayRequests runs a tool loop and returns the captured request bodies.
func replayRequests(t *testing.T, model llm.Model, opts ...Option) []map[string]any {
	t.Helper()
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			bodies = append(bodies, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, replayReply)
		}))
	defer srv.Close()
	client := NewLLM(append([]Option{
		openai.WithAPIKey("k"), openai.WithBaseURL(srv.URL), openai.WithModel(model),
	}, opts...)...)
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
	return bodies
}

// assistantWire returns the assistant message of the last request.
func assistantWire(t *testing.T, bodies []map[string]any) map[string]any {
	t.Helper()
	for _, m := range bodies[len(bodies)-1]["messages"].([]any) {
		if m.(map[string]any)["role"] == "assistant" {
			return m.(map[string]any)
		}
	}
	t.Fatal("no assistant message")
	return nil
}

// TestToolFollowUpKeepsReasoningField verifies the model's own field name is
// reused and the thinking-history mode is sent.
func TestToolFollowUpKeepsReasoningField(t *testing.T) {
	bodies := replayRequests(t, llm.Model{APIModel: "m", CanReason: true},
		WithPreservedThinking(true))
	wire := assistantWire(t, bodies)
	if wire["reasoning"] != "think" {
		t.Errorf("wire = %v", wire)
	}
	if _, ok := wire["reasoning_content"]; ok {
		t.Errorf("wire = %v", wire)
	}
	kwargs, _ := bodies[len(bodies)-1]["chat_template_kwargs"].(map[string]any)
	if kwargs["clear_thinking"] != false {
		t.Errorf("chat_template_kwargs = %v", kwargs)
	}
}
