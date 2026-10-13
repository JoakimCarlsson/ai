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

// TestThinking verifies the provider toggle and capability gate on the wire.
func TestThinking(t *testing.T) {
	for _, state := range []string{"enabled", "non-reasoning", "unset"} {
		t.Run(state, func(t *testing.T) {
			var body map[string]any
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(
						w,
						`{"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":20}}`,
					)
				}),
			)
			defer srv.Close()
			opts := []Option{
				openai.WithAPIKey("test"),
				openai.WithBaseURL(srv.URL),
				openai.WithModel(
					llm.Model{
						APIModel:  "qwen",
						CanReason: state != "non-reasoning",
					},
				),
			}
			if state != "unset" {
				opts = append(opts, WithThinking(false))
			}
			resp, err := NewLLM(
				opts...).SendMessages(context.Background(), []message.Message{message.NewUserMessage("hi")}, nil)
			if err != nil {
				t.Fatal(err)
			}
			reasoning, present := body["reasoning"].(map[string]any)
			if state == "enabled" {
				if !present || reasoning["enabled"] != false {
					t.Errorf("reasoning = %v", body["reasoning"])
				}
			} else if present {
				t.Errorf("unexpected reasoning: %v", reasoning)
			}
			if resp.Usage.ReasoningTokens != 0 {
				t.Errorf(
					"undocumented reasoning count = %d",
					resp.Usage.ReasoningTokens,
				)
			}
		})
	}
}
