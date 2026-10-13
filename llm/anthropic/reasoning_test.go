package anthropic

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

// reasoningTransport redirects Anthropic SDK requests to the test server.
type reasoningTransport struct{ host string }

// RoundTrip rewrites the endpoint while retaining the request body.
func (r reasoningTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = r.host
	return http.DefaultTransport.RoundTrip(req)
}

// TestThinkingUsage verifies explicit disabling and thinking-token usage.
func TestThinkingUsage(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, state := range []string{"enabled", "budget", "non-reasoning", "unset"} {
			t.Run(
				state+map[bool]string{false: "/send", true: "/stream"}[stream],
				func(t *testing.T) {
					var body map[string]any
					reply := `{"id":"msg_x","type":"message","role":"assistant","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":20,"output_tokens_details":{"thinking_tokens":12}}}`
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
										"event: message_start\ndata: "+`{"type":"message_start","message":{"id":"msg_x","type":"message","role":"assistant","content":[],"usage":{"input_tokens":1,"output_tokens":0}}}`+"\n\nevent: message_delta\ndata: "+`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":20,"output_tokens_details":{"thinking_tokens":12}}}`+"\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
									)
								} else {
									w.Header().Set("Content-Type", "application/json")
									_, _ = io.WriteString(w, reply)
								}
							},
						),
					)
					defer srv.Close()
					opts := []Option{
						WithAPIKey("test"),
						WithModel(
							llm.Model{
								APIModel:  "claude-sonnet-4-6",
								CanReason: state != "non-reasoning",
							},
						),
						WithMaxTokens(4096),
						WithHTTPClient(
							&http.Client{
								Transport: reasoningTransport{
									host: srv.Listener.Addr().String(),
								},
							},
						),
					}
					if state != "unset" {
						opts = append(
							opts,
							WithReasoningEffort(ReasoningEffortHigh),
							WithThinkingBudget(0),
						)
					}
					if state == "budget" {
						opts = append(opts, WithThinkingBudget(1024))
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
					thinking, present := body["thinking"].(map[string]any)
					switch state {
					case "enabled":
						if !present || thinking["type"] != "disabled" {
							t.Errorf("thinking = %v", thinking)
						}
					case "budget":
						if thinking["type"] != "enabled" ||
							thinking["budget_tokens"] != float64(1024) {
							t.Errorf("thinking = %v", thinking)
						}
					default:
						if present {
							t.Errorf("unexpected thinking = %v", thinking)
						}
					}
					if _, present := body["output_config"]; present {
						t.Errorf(
							"manual thinking should override adaptive effort: %v",
							body["output_config"],
						)
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
