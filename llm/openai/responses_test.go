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
)

const responsesOK = `{"id":"resp_1","object":"response","status":"completed",` +
	`"output":[{"type":"message","role":"assistant",` +
	`"content":[{"type":"output_text","text":"hi"}]}],` +
	`"usage":{"input_tokens":1,"output_tokens":1}}`

// TestResponsesWithHTTPClientTransportUsed confirms a client injected via
// WithResponsesHTTPClient handles outgoing requests: the wrapped transport's
// counter increments, proving the SDK default client was replaced.
func TestResponsesWithHTTPClientTransportUsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, responsesOK)
		}))
	defer srv.Close()

	var n int
	client := NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-4o-mini"}),
		WithResponsesHTTPClient(&http.Client{
			Transport: countingRT{RoundTripper: http.DefaultTransport, n: &n},
		}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if n == 0 {
		t.Error("injected transport was not used for the request")
	}
}

// newResponsesServer returns a test server that captures the request body into
// capture (when non-nil) and replies with the given Responses-API JSON.
// Mirrors newCompletionServer (openai_test.go) for the Responses request shape.
func newResponsesServer(
	t *testing.T,
	capture *map[string]any,
	response string,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if capture != nil {
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, capture)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, response)
		}))
}

// TestWireReasoningEffortLevels_Responses confirms each of the six
// ReasoningEffort levels maps to the expected reasoning.effort wire value on
// the Responses API path.
func TestWireReasoningEffortLevels_Responses(t *testing.T) {
	tests := []struct {
		name   string
		effort ReasoningEffort
		want   string
	}{
		{"none", ReasoningEffortNone, "none"},
		{"minimal", ReasoningEffortMinimal, "minimal"},
		{"low", ReasoningEffortLow, "low"},
		{"medium", ReasoningEffortMedium, "medium"},
		{"high", ReasoningEffortHigh, "high"},
		{"xhigh", ReasoningEffortXhigh, "xhigh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]any
			srv := newResponsesServer(t, &body, responsesOK)
			defer srv.Close()

			client := NewResponsesLLM(
				WithResponsesAPIKey("test-key"),
				WithResponsesBaseURL(srv.URL),
				WithResponsesModel(
					llm.Model{APIModel: "gpt-5", CanReason: true},
				),
				WithResponsesReasoningEffort(tt.effort),
			)

			if _, err := client.SendMessages(
				context.Background(),
				[]message.Message{
					message.NewUserMessage("hi"),
				},
				nil,
			); err != nil {
				t.Fatalf("SendMessages: %v", err)
			}

			reasoning, ok := body["reasoning"].(map[string]any)
			if !ok {
				t.Fatalf("reasoning = %v (%T), want object",
					body["reasoning"], body["reasoning"])
			}
			got, _ := reasoning["effort"].(string)
			if got != tt.want {
				t.Errorf("reasoning.effort = %v, want %q",
					reasoning["effort"], tt.want)
			}
		})
	}
}

// TestWireReasoningEffortOmittedWhenModelCannotReason_Responses confirms the
// CanReason gate in responsesClient.preparedParams omits the whole reasoning
// object when the model can't reason, for every level.
func TestWireReasoningEffortOmittedWhenModelCannotReason_Responses(
	t *testing.T,
) {
	levels := []ReasoningEffort{
		ReasoningEffortNone, ReasoningEffortMinimal, ReasoningEffortLow,
		ReasoningEffortMedium, ReasoningEffortHigh, ReasoningEffortXhigh,
	}

	for _, effort := range levels {
		t.Run(string(effort), func(t *testing.T) {
			var body map[string]any
			srv := newResponsesServer(t, &body, responsesOK)
			defer srv.Close()

			client := NewResponsesLLM(
				WithResponsesAPIKey("test-key"),
				WithResponsesBaseURL(srv.URL),
				WithResponsesModel(
					llm.Model{APIModel: "gpt-4o-mini", CanReason: false},
				),
				WithResponsesReasoningEffort(effort),
			)

			if _, err := client.SendMessages(
				context.Background(),
				[]message.Message{
					message.NewUserMessage("hi"),
				},
				nil,
			); err != nil {
				t.Fatalf("SendMessages: %v", err)
			}

			if _, present := body["reasoning"]; present {
				t.Errorf(
					"reasoning should be omitted when CanReason=false, got %v",
					body["reasoning"],
				)
			}
		})
	}
}

// TestWireReasoningEffortOmittedWhenUnset_Responses confirms that a
// reasoning-capable model with no WithResponsesReasoningEffort call sends no
// reasoning object on the Responses path -- there is no default effort level
// applied on the caller's behalf.
func TestWireReasoningEffortOmittedWhenUnset_Responses(t *testing.T) {
	var body map[string]any
	srv := newResponsesServer(t, &body, responsesOK)
	defer srv.Close()

	client := NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-5", CanReason: true}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if _, present := body["reasoning"]; present {
		t.Errorf("reasoning should be omitted when unset, got %v",
			body["reasoning"])
	}
}
