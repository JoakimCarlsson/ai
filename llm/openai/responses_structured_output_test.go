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
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/types"
)

// TestResponsesStructuredOutputClosesSchema verifies strict output requests on both paths.
func TestResponsesStructuredOutputClosesSchema(t *testing.T) {
	for _, stream := range []bool{false, true} {
		name := "send"
		if stream {
			name = "stream"
		}
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var body struct {
						Text struct {
							Format struct {
								Strict bool           `json:"strict"`
								Schema map[string]any `json:"schema"`
							} `json:"format"`
						} `json:"text"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if !body.Text.Format.Strict {
						t.Error("structured format must be strict")
					}
					if value, present := body.Text.Format.Schema["additionalProperties"]; !present ||
						value != false {
						t.Errorf(
							"additionalProperties = %v, present = %t; want explicit false",
							value,
							present,
						)
					}
					reply := `{"id":"resp_1","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"answer\":6769}"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = io.WriteString(
							w,
							"event: response.output_text.delta\ndata: "+`{"type":"response.output_text.delta","sequence_number":0,"delta":"{\"answer\":6769}"}`+"\n\nevent: response.completed\ndata: "+`{"type":"response.completed","sequence_number":1,"response":`+reply+"}\n\n",
						)
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, reply)
					}
				}),
			)
			defer srv.Close()
			client := NewLLM(
				WithAPIKey("test"),
				WithBaseURL(srv.URL),
				WithModel(
					llm.Model{
						APIModel:              "gpt-6.1-sol",
						CanReason:             true,
						SupportsStructuredOut: true,
					},
				),
				WithReasoningEffort(ReasoningEffortLow),
			)
			output := schema.NewStructuredOutputInfo(
				"answer",
				"An integer answer",
				map[string]any{"answer": map[string]any{"type": "integer"}},
				[]string{"answer"},
			)
			msgs := []message.Message{
				message.NewUserMessage("Return the answer"),
			}
			var resp *llm.Response
			if stream {
				for event := range client.StreamResponseWithStructuredOutput(context.Background(), msgs, nil, output) {
					if event.Type == types.EventError {
						t.Fatal(event.Error)
					}
					if event.Type == types.EventComplete {
						resp = event.Response
					}
				}
			} else {
				var err error
				resp, err = client.SendMessagesWithStructuredOutput(context.Background(), msgs, nil, output)
				if err != nil {
					t.Fatal(err)
				}
			}
			if resp == nil || resp.StructuredOutput == nil ||
				!resp.UsedNativeStructuredOutput {
				t.Fatal("missing native structured response")
			}
			if *resp.StructuredOutput != `{"answer":6769}` {
				t.Errorf("structured output = %q", *resp.StructuredOutput)
			}
		})
	}
}
