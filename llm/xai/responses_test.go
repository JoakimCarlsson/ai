package xai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

const responsesOK = `{"id":"resp_1","object":"response","status":"completed",` +
	`"output":[{"type":"message","role":"assistant",` +
	`"content":[{"type":"output_text","text":"hi"}]}],` +
	`"usage":{"input_tokens":1,"output_tokens":1}}`

// countingRT is an http.RoundTripper that increments a counter on every request
// before delegating to the wrapped transport, used to prove an injected client's
// transport actually handled the outgoing request.
type countingRT struct {
	http.RoundTripper
	n *int
}

func (c countingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	*c.n++
	return c.RoundTripper.RoundTrip(r)
}

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
		WithResponsesModel(model.Model{APIModel: "grok-4"}),
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
