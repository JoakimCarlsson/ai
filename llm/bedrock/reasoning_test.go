package bedrock

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/llm/anthropic"
	"github.com/joakimcarlsson/ai/message"
)

// reasoningTransport redirects signed Bedrock requests to a local server.
type reasoningTransport struct{ host string }

// RoundTrip changes only the destination of a signed request.
func (r reasoningTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = r.host
	return http.DefaultTransport.RoundTrip(req)
}

// TestReasoningForwarding verifies Claude effort and budget reach Bedrock.
func TestReasoningForwarding(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	for _, state := range []string{"effort", "budget", "disabled", "non-reasoning", "unset"} {
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
						`{"id":"msg_x","type":"message","role":"assistant","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":20,"output_tokens_details":{"thinking_tokens":12}}}`,
					)
				}),
			)
			defer srv.Close()
			opts := []Option{
				WithModel(
					llm.Model{
						APIModel:  "anthropic.claude-sonnet-4-6",
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
					WithReasoningEffort(anthropic.ReasoningEffortLow),
				)
			}
			if state == "budget" {
				opts = append(opts, WithThinkingBudget(1024))
			}
			if state == "disabled" || state == "non-reasoning" {
				opts = append(opts, WithThinkingBudget(0))
			}
			resp, err := NewLLM(
				opts...).SendMessages(context.Background(), []message.Message{message.NewUserMessage("hi")}, nil)
			if err != nil {
				t.Fatal(err)
			}
			thinking, present := body["thinking"].(map[string]any)
			switch state {
			case "effort":
				if thinking["type"] != "adaptive" {
					t.Errorf("thinking = %v", thinking)
				}
				config, _ := body["output_config"].(map[string]any)
				if config["effort"] != "low" {
					t.Errorf("output_config = %v", config)
				}
			case "budget":
				if thinking["type"] != "enabled" ||
					thinking["budget_tokens"] != float64(1024) {
					t.Errorf("thinking = %v", thinking)
				}
			case "disabled":
				if thinking["type"] != "disabled" {
					t.Errorf("thinking = %v", thinking)
				}
			default:
				if present {
					t.Errorf("unexpected thinking = %v", thinking)
				}
			}
			if resp.Usage.ReasoningTokens != 12 ||
				resp.Usage.OutputTokens != 20 {
				t.Errorf("usage = %+v", resp.Usage)
			}
		})
	}
}
