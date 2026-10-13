package perplexity_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/llm/perplexity"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

// TestReasoningUsage verifies top-level reasoning usage on plain and SSE paths.
func TestReasoningUsage(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(
			map[bool]string{false: "send", true: "stream"}[stream],
			func(t *testing.T) {
				var body map[string]any
				srv := httptest.NewServer(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
								t.Error(err)
							}
							usage := `"usage":{"prompt_tokens":1,"completion_tokens":20,"reasoning_tokens":40}`
							if stream {
								w.Header().
									Set("Content-Type", "text/event-stream")
								_, _ = io.WriteString(
									w,
									"data: "+`{"id":"x","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":"stop"}],`+usage+"}\n\ndata: [DONE]\n\n",
								)
							} else {
								w.Header().Set("Content-Type", "application/json")
								_, _ = io.WriteString(w, `{"id":"x","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],`+usage+"}")
							}
						},
					),
				)
				defer srv.Close()
				client := perplexity.NewLLM(
					openai.WithAPIKey("test"),
					openai.WithBaseURL(srv.URL),
					openai.WithModel(
						llm.Model{
							APIModel:  "sonar-deep-research",
							CanReason: true,
						},
					),
					openai.WithReasoningEffort(openai.ReasoningEffortLow),
				)
				msgs := []message.Message{message.NewUserMessage("hi")}
				var resp *llm.Response
				if stream {
					for ev := range client.StreamResponse(context.Background(), msgs, nil) {
						if ev.Type == types.EventError {
							t.Fatal(ev.Error)
						}
						if ev.Type == types.EventComplete {
							resp = ev.Response
						}
					}
				} else {
					var err error
					resp, err = client.SendMessages(context.Background(), msgs, nil)
					if err != nil {
						t.Fatal(err)
					}
				}
				if resp == nil {
					t.Fatal("no response")
				}
				if body["reasoning_effort"] != "low" {
					t.Errorf("effort = %v", body["reasoning_effort"])
				}
				if resp.Usage.ReasoningTokens != 40 ||
					resp.Usage.OutputTokens != 60 {
					t.Errorf("usage = %+v", resp.Usage)
				}
			},
		)
	}
}
