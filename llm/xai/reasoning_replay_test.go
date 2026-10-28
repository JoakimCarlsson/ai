package xai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
)

const replayOutput = `[{"type":"reasoning","id":"rs_1","encrypted_content":"enc1","summary":[{"type":"summary_text","text":"one"}]},` +
	`{"type":"reasoning","id":"rs_2","encrypted_content":"enc2","summary":[]},` +
	`{"type":"function_call","id":"fc_1","call_id":"call_1","name":"f","arguments":"{}","status":"completed"}]`

const replayBody = `{"id":"resp_1","object":"response","status":"completed","output":` +
	replayOutput + `,"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`

// replayServer captures request bodies and replies with a stateless response,
// as JSON, or as the equivalent event stream when the request asks to stream.
func replayServer(
	t *testing.T,
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
			if body["stream"] != true {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, replayBody)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			var items []json.RawMessage
			_ = json.Unmarshal([]byte(replayOutput), &items)
			seq := 0
			for _, item := range items {
				added := string(item)
				if bytes.Contains(item, []byte(`"reasoning"`)) {
					added = `{"type":"reasoning","id":"x","encrypted_content":"partial","summary":[]}`
				}
				_, _ = io.WriteString(w, "data: "+`{"type":"response.output_item.added","sequence_number":`+
					string(rune('0'+seq))+`,"output_index":0,"item":`+added+`}`+"\n\n")
				_, _ = io.WriteString(w, "data: "+`{"type":"response.output_item.done","sequence_number":`+
					string(rune('0'+seq))+`,"output_index":0,"item":`+string(item)+`}`+"\n\n")
				seq++
			}
			_, _ = io.WriteString(w, "data: "+`{"type":"response.completed","sequence_number":9,"response":`+replayBody+`}`+"\n\n")
		}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}

// TestStatelessReasoningReplay verifies the request flags, retained items and
// replay order after a tool result, for both response paths.
func TestStatelessReasoningReplay(t *testing.T) {
	for _, stream := range []bool{false, true} {
		name := map[bool]string{false: "send", true: "stream"}[stream]
		t.Run(name, func(t *testing.T) {
			srv, bodies := replayServer(t)
			client := NewLLM(
				WithAPIKey("k"),
				WithBaseURL(srv.URL),
				WithModel(llm.Model{APIModel: "grok", CanReason: true}),
				WithStore(false),
				WithWebSearch(),
			)
			history := []message.Message{message.NewUserMessage("go")}
			var resp *llm.Response
			if stream {
				for ev := range client.StreamResponse(context.Background(), history, nil) {
					if ev.Response != nil {
						resp = ev.Response
					}
				}
			} else {
				var err error
				resp, err = client.SendMessages(context.Background(), history, nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			first := (*bodies)[0]
			if first["store"] != false ||
				!reflect.DeepEqual(first["include"], []any{"reasoning.encrypted_content"}) {
				t.Errorf("request = %v", first)
			}
			if _, ok := first["previous_response_id"]; ok {
				t.Error("previous_response_id sent")
			}
			if len(first["tools"].([]any)) != 1 {
				t.Errorf("tools = %v", first["tools"])
			}
			if len(resp.ReasoningItems) != 2 ||
				resp.ReasoningItems[0].EncryptedContent != "enc1" ||
				resp.ReasoningItems[1].EncryptedContent != "enc2" {
				t.Fatalf("items = %+v", resp.ReasoningItems)
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
			input := (*bodies)[len(*bodies)-1]["input"].([]any)
			var types []string
			var ids []string
			for _, in := range input {
				m := in.(map[string]any)
				kind, _ := m["type"].(string)
				if kind == "" {
					kind = "message"
				}
				types = append(types, kind)
				if kind == "reasoning" {
					ids = append(ids, m["id"].(string))
				}
			}
			want := []string{"message", "reasoning", "reasoning", "function_call", "function_call_output"}
			if !reflect.DeepEqual(types, want) ||
				!reflect.DeepEqual(ids, []string{"rs_1", "rs_2"}) {
				t.Errorf("input types = %v ids = %v", types, ids)
			}
		})
	}
}

// TestStoreUnsetSendsNothing verifies the default request is unchanged.
func TestStoreUnsetSendsNothing(t *testing.T) {
	srv, bodies := replayServer(t)
	client := NewLLM(
		WithAPIKey("k"),
		WithBaseURL(srv.URL),
		WithModel(llm.Model{APIModel: "grok", CanReason: true}),
	)
	resp, err := client.SendMessages(
		context.Background(),
		[]message.Message{message.NewUserMessage("go")},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"store", "include"} {
		if _, ok := (*bodies)[0][key]; ok {
			t.Errorf("%s sent", key)
		}
	}
	if len(resp.ReasoningItems) != 0 {
		t.Errorf("items = %+v", resp.ReasoningItems)
	}
}
