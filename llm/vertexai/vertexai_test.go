package vertexai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

const generateContentOK = `{"candidates":[{"content":{"role":"model",` +
	`"parts":[{"text":"hi"}]},"finishReason":"STOP"}],` +
	`"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1,` +
	`"totalTokenCount":2}}`

// redirectRT rewrites every request to point at the test server's host before
// delegating to the wrapped transport, and counts the requests it handled. The
// genai SDK exposes no base-URL option here, so redirecting at the transport is
// how a test reaches a local httptest server through an injected client. Setting
// HTTPClient also makes the genai client skip Application Default Credentials, so
// no real GCP auth is needed.
type redirectRT struct {
	base http.RoundTripper
	host string
	n    *int
}

func (c redirectRT) RoundTrip(r *http.Request) (*http.Response, error) {
	*c.n++
	r.URL.Scheme = "http"
	r.URL.Host = c.host
	return c.base.RoundTrip(r)
}

// TestWithHTTPClientTransportUsed confirms a client injected via WithHTTPClient
// handles outgoing requests: the wrapped transport's counter increments, proving
// the SDK default client was replaced.
func TestWithHTTPClientTransportUsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, generateContentOK)
		}))
	defer srv.Close()

	var n int
	client := NewLLM(
		WithProject("test-project"),
		WithLocation("us-central1"),
		WithModel(model.Model{APIModel: "gemini-2.0-flash"}),
		WithHTTPClient(&http.Client{
			Transport: redirectRT{
				base: http.DefaultTransport,
				host: srv.Listener.Addr().String(),
				n:    &n,
			},
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
