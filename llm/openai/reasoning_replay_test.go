package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
)

const detailsJSON = `[{"type":"reasoning.text","text":"a","signature":"sig","index":0},` +
	`{"type":"reasoning.encrypted","data":"blob","index":1}]`

// captureServer records request bodies and replies with the given payloads in
// order, repeating the last one.
func captureServer(
	t *testing.T,
	contentType string,
	replies ...string,
) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			bodies = append(bodies, body)
			reply := replies[min(len(bodies), len(replies))-1]
			w.Header().Set("Content-Type", contentType)
			_, _ = io.WriteString(w, reply)
		}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}

// roundTrip serializes a message through JSON as session storage would.
func roundTrip(t *testing.T, msg message.Message) message.Message {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var out message.Message
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// assistantFrom builds the history message agents keep for a response.
func assistantFrom(resp *llm.Response) message.Message {
	msg := message.NewAssistantMessage()
	for _, item := range resp.ReasoningItems {
		msg.Parts = append(msg.Parts, item)
	}
	if resp.Content != "" {
		msg.AppendContent(resp.Content)
	}
	msg.AppendToolCalls(resp.ToolCalls)
	return msg
}

// followUp returns the history for the request after a tool result.
func followUp(assistant message.Message) []message.Message {
	result := message.NewMessage(message.Tool, nil)
	result.AddToolResult(message.ToolResult{ToolCallID: "call_1", Content: "ok"})
	return []message.Message{
		message.NewUserMessage("go"),
		assistant,
		result,
	}
}

// assistantWire returns the assistant message of the last captured request.
func assistantWire(t *testing.T, bodies []map[string]any) map[string]any {
	t.Helper()
	msgs := bodies[len(bodies)-1]["messages"].([]any)
	for _, m := range msgs {
		if m.(map[string]any)["role"] == "assistant" {
			return m.(map[string]any)
		}
	}
	t.Fatal("no assistant message")
	return nil
}

const toolReply = `{"choices":[{"index":0,"message":{"role":"assistant","content":"",` +
	`"reasoning":"think","reasoning_details":` + detailsJSON + `,` +
	`"tool_calls":[{"id":"call_1","type":"function","function":{"name":"f","arguments":"{}"}}]},` +
	`"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`

// TestChatReasoningReplay verifies the field name and details survive JSON
// storage and are replayed unchanged after a tool result.
func TestChatReasoningReplay(t *testing.T) {
	cases := map[string][]Option{
		"details": {WithReasoningDetailsReplay(true)},
		"text":    {WithReasoningContentReplay(true)},
		"off":     nil,
		"explicit off overrides default": {
			WithReasoningReplayDefault(func(llm.Model) bool { return true }),
			WithReasoningContentReplay(false),
		},
		"default on": {
			WithReasoningReplayDefault(func(llm.Model) bool { return true }),
		},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			srv, bodies := captureServer(t, "application/json", toolReply)
			client := NewChatCompletionsLLM(append([]Option{
				WithAPIKey("k"), WithBaseURL(srv.URL),
				WithModel(llm.Model{APIModel: "m", CanReason: true}),
			}, opts...)...)
			resp, err := client.SendMessages(
				context.Background(),
				[]message.Message{message.NewUserMessage("go")},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got := resp.ReasoningItems[0]; got.Field != "reasoning" ||
				string(got.Details) != detailsJSON {
				t.Fatalf("item = %+v", got)
			}
			assistant := roundTrip(t, assistantFrom(resp))
			if _, err := client.SendMessages(
				context.Background(), followUp(assistant), nil,
			); err != nil {
				t.Fatal(err)
			}
			wire := assistantWire(t, *bodies)
			_, hasDetails := wire["reasoning_details"]
			_, hasText := wire["reasoning"]
			_, hasContent := wire["reasoning_content"]
			switch name {
			case "details":
				raw, _ := json.Marshal(wire["reasoning_details"])
				var want any
				_ = json.Unmarshal([]byte(detailsJSON), &want)
				wantRaw, _ := json.Marshal(want)
				if string(raw) != string(wantRaw) || hasText || hasContent {
					t.Errorf("wire = %v", wire)
				}
			case "text", "default on":
				if wire["reasoning"] != "think" || hasDetails || hasContent {
					t.Errorf("wire = %v", wire)
				}
			default:
				if hasDetails || hasText || hasContent {
					t.Errorf("wire = %v", wire)
				}
			}
		})
	}
}

// TestChatStreamReasoningDetails verifies streamed fragments merge into the
// complete ordered array and the reasoning field name is kept.
func TestChatStreamReasoningDetails(t *testing.T) {
	sse := "data: " + `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"th","reasoning_details":[{"type":"reasoning.text","text":"a","index":0}]}}]}` + "\n\n" +
		"data: " + `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"reasoning_content":"ink","reasoning_details":[{"type":"reasoning.text","text":"b","signature":"sig","index":0},{"type":"reasoning.encrypted","data":"blob","index":1}]}}]}` + "\n\n" +
		"data: " + `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":"stop"}]}` + "\n\n" +
		"data: [DONE]\n\n"
	srv, _ := captureServer(t, "text/event-stream", sse)
	client := NewChatCompletionsLLM(WithAPIKey("k"), WithBaseURL(srv.URL),
		WithModel(llm.Model{APIModel: "m"}))
	var final *llm.Response
	for ev := range client.StreamResponse(
		context.Background(),
		[]message.Message{message.NewUserMessage("go")},
		nil,
	) {
		if ev.Response != nil {
			final = ev.Response
		}
	}
	if final == nil || len(final.ReasoningItems) != 1 {
		t.Fatalf("response = %+v", final)
	}
	item := final.ReasoningItems[0]
	if item.Text != "think" || item.Field != "reasoning_content" {
		t.Errorf("item = %+v", item)
	}
	var details []map[string]any
	if err := json.Unmarshal(item.Details, &details); err != nil {
		t.Fatal(err)
	}
	if len(details) != 2 || details[0]["text"] != "ab" ||
		details[0]["signature"] != "sig" || details[1]["data"] != "blob" {
		t.Errorf("details = %v", details)
	}
}
