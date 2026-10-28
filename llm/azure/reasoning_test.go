package azure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
)

// TestReasoningEffort verifies Azure routing forwards effort and reasoning usage.
func TestReasoningEffort(t *testing.T) {
	for _, path := range []string{"", "/openai/v1"} {
		for _, state := range []string{"enabled", "non-reasoning", "unset"} {
			t.Run(path+"/"+state, func(t *testing.T) {
				var body map[string]any
				srv := httptest.NewTLSServer(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
								t.Error(err)
							}
							w.Header().Set("Content-Type", "application/json")
							_, _ = io.WriteString(
								w,
								`{"id":"x","object":"response","status":"completed","output":[{"type":"message","id":"m","role":"assistant","status":"completed","content":[{"type":"output_text","text":"hi","annotations":[]}]}],"usage":{"input_tokens":1,"output_tokens":20,"total_tokens":21,"input_tokens_details":{"cached_tokens":0},"output_tokens_details":{"reasoning_tokens":12}}}`,
							)
						},
					),
				)
				defer srv.Close()
				opts := []Option{
					WithAPIKey("test"),
					WithEndpoint(srv.URL + path),
					WithAPIVersion("2025-04-01-preview"),
					WithDeployment("custom"),
					WithHTTPClient(srv.Client()),
					WithReasoning(state != "non-reasoning"),
				}
				if state != "unset" {
					opts = append(
						opts,
						WithReasoningEffort(openai.ReasoningEffortNone),
					)
				}
				resp, err := NewLLM(
					opts...).SendMessages(context.Background(), []message.Message{message.NewUserMessage("hi")}, nil)
				if err != nil {
					t.Fatal(err)
				}
				reasoning, _ := body["reasoning"].(map[string]any)
				effort, present := reasoning["effort"]
				if state == "enabled" {
					if effort != "none" {
						t.Errorf("effort = %v, want none", effort)
					}
				} else if present {
					t.Errorf("unexpected effort: %v", effort)
				}
				if resp.Usage.ReasoningTokens != 12 ||
					resp.Usage.OutputTokens != 20 {
					t.Errorf("usage = %+v", resp.Usage)
				}
			})
		}
	}
}
