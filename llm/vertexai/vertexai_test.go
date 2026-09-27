package vertexai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/llm"

	"github.com/joakimcarlsson/ai/message"
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

// RoundTrip counts the request, points it at the test server and delegates it
// to the wrapped transport.
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
		WithModel(llm.Model{APIModel: "gemini-2.0-flash"}),
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

// capturingClient builds a Vertex AI client whose requests are decoded into
// body and answered with reply.
func capturingClient(
	t *testing.T,
	body *map[string]any,
	reply string,
	opts ...Option,
) llm.LLM {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, reply)
		}))
	t.Cleanup(srv.Close)

	var n int
	return NewLLM(append([]Option{
		WithProject("test-project"),
		WithLocation("us-central1"),
		WithModel(llm.Model{APIModel: "gemini-2.0-flash"}),
		WithHTTPClient(&http.Client{
			Transport: redirectRT{
				base: http.DefaultTransport,
				host: srv.Listener.Addr().String(),
				n:    &n,
			},
		}),
	}, opts...)...)
}

// TestWithCachedContentReachesTheRequest confirms WithCachedContent is
// forwarded to the Gemini client and sent on generateContent.
func TestWithCachedContentReachesTheRequest(t *testing.T) {
	var body map[string]any
	client := capturingClient(t, &body, generateContentOK,
		WithCachedContent("cachedContents/x"))

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	want := "projects/test-project/locations/us-central1/cachedContents/x"
	if got := body["cachedContent"]; got != want {
		t.Errorf("cachedContent = %v, want %q", got, want)
	}
}

// TestCreateCacheIsReachableAndSendsTheTTL confirms the Vertex AI client
// exposes CreateCache through the traced llm.LLM and forwards WithCacheTTL.
func TestCreateCacheIsReachableAndSendsTheTTL(t *testing.T) {
	var body map[string]any
	client := capturingClient(t, &body, `{"name":"cachedContents/abc"}`,
		WithCacheTTL(90*time.Second))

	ccp, ok := client.(llm.ContextCacheProvider)
	if !ok {
		t.Fatal("NewLLM result does not implement llm.ContextCacheProvider")
	}
	name, err := ccp.CreateCache(context.Background(),
		[]message.Message{message.NewUserMessage("remember this")}, nil)
	if err != nil {
		t.Fatalf("CreateCache: %v", err)
	}

	if name != "cachedContents/abc" {
		t.Errorf("name = %q, want %q", name, "cachedContents/abc")
	}
	if got := body["ttl"]; got != "90s" {
		t.Errorf("ttl = %v, want %q", got, "90s")
	}
}
