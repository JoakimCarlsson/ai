package vertexai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

// TestThinkingUsage verifies budgets and inclusive output usage on both paths.
func TestThinkingUsage(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, state := range []string{"enabled", "non-reasoning", "unset"} {
			t.Run(
				state+map[bool]string{false: "/send", true: "/stream"}[stream],
				func(t *testing.T) {
					var body map[string]any
					reply := `{"candidates":[{"content":{"role":"model","parts":[{"text":"hi"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":8,"thoughtsTokenCount":12,"totalTokenCount":21}}`
					srv := httptest.NewServer(
						http.HandlerFunc(
							func(w http.ResponseWriter, r *http.Request) {
								if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
									t.Error(err)
								}
								if stream {
									w.Header().
										Set("Content-Type", "text/event-stream")
									_, _ = io.WriteString(
										w,
										"data: "+reply+"\n\n",
									)
								} else {
									w.Header().Set("Content-Type", "application/json")
									_, _ = io.WriteString(w, reply)
								}
							},
						),
					)
					defer srv.Close()
					var n int
					opts := []Option{
						WithProject("test"),
						WithLocation("us-central1"),
						WithModel(
							llm.Model{
								APIModel:  "gemini-2.5-flash",
								CanReason: state != "non-reasoning",
							},
						),
						WithHTTPClient(
							&http.Client{
								Transport: redirectRT{
									base: http.DefaultTransport,
									host: srv.Listener.Addr().String(),
									n:    &n,
								},
							},
						),
					}
					if state != "unset" {
						opts = append(opts, WithThinkingBudget(0))
					}
					client := NewLLM(opts...)
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
					config, _ := body["generationConfig"].(map[string]any)
					thinking, present := config["thinkingConfig"].(map[string]any)
					if state == "enabled" {
						if !present ||
							thinking["thinkingBudget"] != float64(0) {
							t.Errorf("thinkingConfig = %v", thinking)
						}
					} else if present {
						t.Errorf("unexpected thinkingConfig = %v", thinking)
					}
					if resp.Usage.ReasoningTokens != 12 ||
						resp.Usage.OutputTokens != 20 {
						t.Errorf("usage = %+v", resp.Usage)
					}
				},
			)
		}
	}
}
