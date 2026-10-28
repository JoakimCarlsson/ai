package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/types"
)

const encryptedReasoning = `{"type":"reasoning","id":"rs_1","encrypted_content":"opaque-state","summary":[{"type":"summary_text","text":"Plan"},{"type":"summary_text","text":"Act"}]}`

const reasoningMessage = `{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"hi"}]}`

const reasoningCall = `{"type":"function_call","id":"fc_1","call_id":"call_1","name":"lookup","arguments":"{}","status":"completed"}`

// TestResponsesStoreWire checks explicit storage and the reasoning include gate.
func TestResponsesStoreWire(t *testing.T) {
	for _, tc := range []struct {
		name      string
		store     *bool
		canReason bool
	}{
		{name: "default", canReason: true},
		{name: "stored", store: boolPointer(true), canReason: true},
		{name: "stateless reasoning", store: boolPointer(false), canReason: true},
		{name: "stateless ordinary", store: boolPointer(false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			server := newResponsesServer(t, &body, responsesOK)
			defer server.Close()
			opts := []Option{
				WithAPIKey("test"), WithBaseURL(server.URL),
				WithModel(
					llm.Model{APIModel: "model", CanReason: tc.canReason},
				),
			}
			if tc.store != nil {
				opts = append(opts, WithStore(*tc.store))
			}
			if _, err := NewLLM(opts...).SendMessages(context.Background(), nil, nil); err != nil {
				t.Fatal(err)
			}
			value, present := body["store"]
			if present != (tc.store != nil) || (present && value != *tc.store) {
				t.Fatalf("store = %v (present %v)", value, present)
			}
			_, included := body["include"]
			wantInclude := tc.store != nil && !*tc.store && tc.canReason
			if included != wantInclude {
				t.Fatalf(
					"include = %v, want present %v",
					body["include"],
					wantInclude,
				)
			}
			if wantInclude &&
				!reflect.DeepEqual(
					body["include"],
					[]any{"reasoning.encrypted_content"},
				) {
				t.Fatalf("include = %v", body["include"])
			}
		})
	}
}

// boolPointer returns an addressable storage choice for table-driven cases.
func boolPointer(value bool) *bool { return &value }

// TestResponsesReasoningReplay verifies wire replay after persisted message
// history, including tool results, streaming completion and structured output.
func TestResponsesReasoningReplay(t *testing.T) {
	for _, mode := range []string{"plain", "tool", "stream", "stream tool", "stream done", "structured", "stream structured"} {
		t.Run(mode, func(t *testing.T) {
			output := reasoningMessage
			toolCall := mode == "tool" || mode == "stream tool"
			if toolCall {
				output = reasoningCall
			}
			payload := `{"id":"resp_1","object":"response","status":"completed","output":[` + encryptedReasoning + `,` + output + `],"usage":{"input_tokens":1,"output_tokens":1}}`
			var bodies []map[string]any
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					bodies = append(bodies, body)
					if body["stream"] == true {
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprintf(
							w,
							"data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"reasoning\",\"id\":\"rs_1\",\"encrypted_content\":\"partial\"}}\n\n",
						)
						fmt.Fprintf(
							w,
							"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":%s}\n\n",
							encryptedReasoning,
						)
						if toolCall {
							fmt.Fprintf(
								w,
								"data: {\"type\":\"response.output_item.added\",\"output_index\":1,\"item\":%s}\n\n",
								reasoningCall,
							)
							fmt.Fprint(
								w,
								"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\"{}\"}\n\n",
							)
						} else {
							fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\n")
						}
						if mode == "stream done" {
							payload = `{"id":"resp_1","status":"completed","output":[],"usage":{}}`
						}
						fmt.Fprintf(
							w,
							"data: {\"type\":\"response.completed\",\"response\":%s}\n\n",
							payload,
						)
					} else {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, payload)
					}
				}),
			)
			defer server.Close()
			client := NewLLM(
				WithAPIKey("test"),
				WithBaseURL(server.URL),
				WithModel(
					llm.Model{APIModel: "reasoner", CanReason: true},
				),
				WithStore(false),
			)
			history := []message.Message{message.NewUserMessage("hello")}
			var response *llm.Response
			var err error
			switch mode {
			case "stream", "stream tool", "stream done", "stream structured":
				var events <-chan llm.Event
				if mode == "stream structured" {
					events = client.StreamResponseWithStructuredOutput(
						context.Background(),
						history,
						nil,
						&schema.StructuredOutputInfo{},
					)
				} else {
					events = client.StreamResponse(context.Background(), history, nil)
				}
				for event := range events {
					if event.Type == types.EventError {
						t.Fatal(event.Error)
					}
					if event.Type == types.EventComplete {
						response = event.Response
					}
				}
			case "structured":
				response, err = client.SendMessagesWithStructuredOutput(
					context.Background(),
					history,
					nil,
					&schema.StructuredOutputInfo{},
				)
			default:
				response, err = client.SendMessages(
					context.Background(),
					history,
					nil,
				)
			}
			if err != nil {
				t.Fatal(err)
			}
			if response == nil || len(response.ReasoningItems) != 1 {
				t.Fatalf("response = %+v", response)
			}
			item := response.ReasoningItems[0]
			if item.ID != "rs_1" || item.EncryptedContent != "opaque-state" ||
				!reflect.DeepEqual(item.Summary, []string{"Plan", "Act"}) {
				t.Fatalf("reasoning = %+v", item)
			}
			assistant := message.NewAssistantMessage()
			assistant.Parts = append(assistant.Parts, item)
			assistant.AppendContent(response.Content)
			assistant.AppendToolCalls(response.ToolCalls)
			history = append(history, assistant)
			if toolCall {
				result := message.Message{Role: message.Tool}
				result.AddToolResult(
					message.ToolResult{ToolCallID: "call_1", Content: "found"},
				)
				history = append(history, result)
			} else {
				history = append(history, message.NewUserMessage("continue"))
			}
			encoded, err := json.Marshal(history)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &history); err != nil {
				t.Fatal(err)
			}
			if _, err := client.SendMessages(context.Background(), history, nil); err != nil {
				t.Fatal(err)
			}
			body := bodies[len(bodies)-1]
			if body["store"] != false ||
				!reflect.DeepEqual(
					body["include"],
					[]any{"reasoning.encrypted_content"},
				) {
				t.Fatalf("storage params = %v", body)
			}
			if _, present := body["previous_response_id"]; present {
				t.Fatal("unexpected server-side state")
			}
			input := body["input"].([]any)
			replayed := input[1].(map[string]any)
			if replayed["type"] != "reasoning" || replayed["id"] != "rs_1" ||
				replayed["encrypted_content"] != "opaque-state" {
				t.Fatalf("input = %v", input)
			}
			if !reflect.DeepEqual(
				replayed["summary"],
				[]any{
					map[string]any{"type": "summary_text", "text": "Plan"},
					map[string]any{"type": "summary_text", "text": "Act"},
				},
			) {
				t.Fatalf("summary = %v", replayed["summary"])
			}
			if toolCall {
				if input[2].(map[string]any)["type"] != "function_call" ||
					input[3].(map[string]any)["type"] != "function_call_output" {
					t.Fatalf("tool input = %v", input)
				}
			} else if input[2].(map[string]any)["role"] != "assistant" {
				t.Fatalf("assistant input = %v", input)
			}
		})
	}
}

// TestResponsesUnencryptedReasoningIsDropped verifies stored IDs and legacy
// reasoning text never become invalid stateless reasoning input items.
func TestResponsesUnencryptedReasoningIsDropped(t *testing.T) {
	var body map[string]any
	server := newResponsesServer(t, &body, responsesOK)
	defer server.Close()
	client := NewLLM(
		WithAPIKey("test"),
		WithBaseURL(server.URL),
		WithStore(false),
	)
	assistant := message.NewAssistantMessage()
	assistant.Parts = append(
		assistant.Parts,
		message.ReasoningContent{ID: "stored", Text: "legacy"},
	)
	assistant.AppendContent("hi")
	if _, err := client.SendMessages(context.Background(), []message.Message{assistant}, nil); err != nil {
		t.Fatal(err)
	}
	input := body["input"].([]any)
	if len(input) != 1 || input[0].(map[string]any)["role"] != "assistant" {
		t.Fatalf("input = %v", input)
	}
}

// TestResponsesReplayEmptySummary verifies the required summary array is sent
// even when the encrypted reasoning item has no visible summary text.
func TestResponsesReplayEmptySummary(t *testing.T) {
	var body map[string]any
	server := newResponsesServer(t, &body, responsesOK)
	defer server.Close()
	client := NewLLM(
		WithAPIKey("test"),
		WithBaseURL(server.URL),
		WithStore(false),
	)
	assistant := message.NewAssistantMessage()
	assistant.Parts = append(assistant.Parts, message.ReasoningContent{
		ID: "rs_1", EncryptedContent: "opaque",
	})
	assistant.AppendContent("hi")
	if _, err := client.SendMessages(context.Background(), []message.Message{assistant}, nil); err != nil {
		t.Fatal(err)
	}
	input := body["input"].([]any)
	item := input[0].(map[string]any)
	if !reflect.DeepEqual(item["summary"], []any{}) {
		t.Fatalf("summary = %#v, want []", item["summary"])
	}
}

// TestResponsesStructuredSchemaRejectsExtraProperties verifies the strict
// schema constraint required by Azure on both Responses request paths.
func TestResponsesStructuredSchemaRejectsExtraProperties(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%t", streaming), func(t *testing.T) {
			var body map[string]any
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if streaming {
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprintf(
							w,
							"data: {\"type\":\"response.completed\",\"response\":%s}\n\n",
							responsesOK,
						)
					} else {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, responsesOK)
					}
				}),
			)
			defer server.Close()
			client := NewLLM(
				WithAPIKey("test"),
				WithBaseURL(server.URL),
				WithStore(false),
			)
			outputSchema := schema.NewStructuredOutputFromStruct(
				"answer",
				"An answer",
				struct {
					Answer int `json:"answer"`
				}{},
			)
			if streaming {
				for event := range client.StreamResponseWithStructuredOutput(context.Background(), nil, nil, outputSchema) {
					if event.Type == types.EventError {
						t.Fatal(event.Error)
					}
				}
			} else if _, err := client.SendMessagesWithStructuredOutput(context.Background(), nil, nil, outputSchema); err != nil {
				t.Fatal(err)
			}
			text := body["text"].(map[string]any)
			format := text["format"].(map[string]any)
			sentSchema := format["schema"].(map[string]any)
			if format["strict"] != true ||
				sentSchema["additionalProperties"] != false {
				t.Fatalf("strict format = %v", format)
			}
		})
	}
}
