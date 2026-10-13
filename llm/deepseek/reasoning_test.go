package deepseek

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

// TestThinking verifies DeepSeek disables thinking only for reasoning models.
func TestThinking(t *testing.T) {
	for _, state := range []string{"enabled", "low", "high", "max", "non-reasoning", "unset"} {
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
						`{"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":20,"completion_tokens_details":{"reasoning_tokens":12}}}`,
					)
				}),
			)
			defer srv.Close()
			opts := []Option{
				openai.WithAPIKey("test"),
				openai.WithBaseURL(srv.URL),
				openai.WithModel(
					llm.Model{
						APIModel:  "deepseek-chat",
						CanReason: state != "non-reasoning",
					},
				),
			}
			if state != "unset" {
				effort := ReasoningEffortNone
				if state == "low" || state == "high" || state == "max" {
					effort = ReasoningEffort(state)
				}
				opts = append(opts, WithReasoningEffort(effort))
			}
			resp, err := NewLLM(
				opts...).SendMessages(context.Background(), []message.Message{message.NewUserMessage("hi")}, nil)
			if err != nil {
				t.Fatal(err)
			}
			reasoning, present := body["thinking"].(map[string]any)
			switch state {
			case "enabled":
				if !present || reasoning["type"] != "disabled" {
					t.Errorf("reasoning = %v", body["thinking"])
				}
			case "low", "high", "max":
				if reasoning["type"] != "enabled" || body["reasoning_effort"] != state {
					t.Errorf("thinking = %v, effort = %v", reasoning, body["reasoning_effort"])
				}
			default:
				if present {
					t.Errorf("unexpected reasoning: %v", reasoning)
				}
			}
			if resp.Usage.ReasoningTokens != 12 {
				t.Errorf("reasoning count = %d", resp.Usage.ReasoningTokens)
			}
		})
	}
}
