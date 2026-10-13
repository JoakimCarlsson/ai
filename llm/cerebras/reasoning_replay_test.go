package cerebras

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

// TestQwenReplaysReasoning verifies Qwen replays reasoning with clear_thinking
// and other families receive neither.
func TestQwenReplaysReasoning(t *testing.T) {
	bodies := replayRequests(t, llm.Model{APIModel: "qwen-3.8-27b", CanReason: true},
		WithClearThinking(false))
	wire := assistantWire(t, bodies)
	if wire["reasoning"] != "think" {
		t.Errorf("wire = %v", wire)
	}
	if bodies[len(bodies)-1]["clear_thinking"] != false {
		t.Errorf("clear_thinking = %v", bodies[len(bodies)-1]["clear_thinking"])
	}
	bodies = replayRequests(t, llm.Model{APIModel: "gpt-oss-120b", CanReason: true},
		WithClearThinking(true))
	wire = assistantWire(t, bodies)
	if _, ok := wire["reasoning"]; ok {
		t.Errorf("wire = %v", wire)
	}
	if _, ok := bodies[len(bodies)-1]["clear_thinking"]; ok {
		t.Error("clear_thinking sent to non-Qwen model")
	}
}
