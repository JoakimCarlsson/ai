package xai

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

// TestResponsesEffort verifies all xAI effort levels and capability gating.
func TestResponsesEffort(t *testing.T) {
	for _, effort := range []ReasoningEffort{ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh, ReasoningEffortXhigh, ""} {
		for _, canReason := range []bool{true, false} {
			t.Run(
				string(
					effort,
				)+map[bool]string{true: "/reasoning", false: "/ordinary"}[canReason],
				func(t *testing.T) {
					var body map[string]any
					srv := httptest.NewServer(
						http.HandlerFunc(
							func(w http.ResponseWriter, r *http.Request) {
								if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
									t.Error(err)
								}
								w.Header().
									Set("Content-Type", "application/json")
								_, _ = io.WriteString(w, responsesCachedBody)
							},
						),
					)
					defer srv.Close()
					opts := []Option{
						WithAPIKey("test"),
						WithBaseURL(srv.URL),
						WithModel(
							llm.Model{
								APIModel:  "grok-4.7",
								CanReason: canReason,
							},
						),
					}
					if effort != "" {
						opts = append(
							opts,
							WithReasoningEffort(effort),
						)
					}
					resp, err := NewLLM(
						opts...).SendMessages(context.Background(), []message.Message{message.NewUserMessage("hi")}, nil)
					if err != nil {
						t.Fatal(err)
					}
					reasoning, present := body["reasoning"].(map[string]any)
					if canReason && effort != "" {
						if reasoning["effort"] != string(effort) {
							t.Errorf("reasoning = %v", reasoning)
						}
					} else if present {
						t.Errorf("unexpected reasoning = %v", reasoning)
					}
					checkCachedUsage(t, resp.Usage)
				},
			)
		}
	}
}
