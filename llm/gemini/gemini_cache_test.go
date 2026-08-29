package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"

	"google.golang.org/genai"
)

// TestWithCachedContentSetsConfig verifies WithCachedContent populates
// config.CachedContent and suppresses inline SystemInstruction/Tools (genai
// rejects a cache combined with either).
func TestWithCachedContentSetsConfig(t *testing.T) {
	c := &Client{}
	WithCachedContent("cachedContents/x")(&c.options)

	cfg := c.buildConfig([]string{"you are a helpful assistant"}, nil)

	if cfg.CachedContent != "cachedContents/x" {
		t.Errorf(
			"CachedContent = %q, want %q",
			cfg.CachedContent,
			"cachedContents/x",
		)
	}
	if cfg.SystemInstruction != nil {
		t.Error("expected SystemInstruction to be nil when a cache is attached")
	}
	if cfg.Tools != nil {
		t.Error("expected Tools to be nil when a cache is attached")
	}
}

// TestWithCachedContentDisabledLeavesEmpty verifies WithDisableCache overrides a
// configured cache: CachedContent stays empty and the inline system
// instruction is still emitted.
func TestWithCachedContentDisabledLeavesEmpty(t *testing.T) {
	c := &Client{}
	WithCachedContent("cachedContents/x")(&c.options)
	WithDisableCache()(&c.options)

	cfg := c.buildConfig([]string{"you are a helpful assistant"}, nil)

	if cfg.CachedContent != "" {
		t.Errorf(
			"CachedContent = %q, want empty when cache disabled",
			cfg.CachedContent,
		)
	}
	if cfg.SystemInstruction == nil {
		t.Error("expected SystemInstruction to be set when cache is disabled")
	}
}

// TestBuildConfigNoCacheLeavesEmpty verifies the default path leaves
// CachedContent empty.
func TestBuildConfigNoCacheLeavesEmpty(t *testing.T) {
	cfg := (&Client{options: Options{model: llm.Model{}}}).buildConfig(nil, nil)
	if cfg.CachedContent != "" {
		t.Errorf("CachedContent = %q, want empty by default", cfg.CachedContent)
	}
}

// cacheClient builds a *Client (not the traced llm.LLM NewLLM returns, since
// CreateCache is not on that interface) whose requests are captured into body
// and answered with status/reply.
func cacheClient(
	t *testing.T,
	body *map[string]any,
	status int,
	reply string,
	opts ...Option,
) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, reply)
		}))
	t.Cleanup(srv.Close)

	options := Options{model: llm.Model{APIModel: "gemini-2.0-flash"}}
	for _, o := range opts {
		o(&options)
	}

	var n int
	gc, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  "test-key",
		Backend: genai.BackendGeminiAPI,
		HTTPClient: &http.Client{
			Transport: rewriteBody{
				base: redirectRT{
					base: http.DefaultTransport,
					host: srv.Listener.Addr().String(),
					n:    &n,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("genai.NewClient: %v", err)
	}
	return &Client{options: options, client: gc}
}

// rewriteBody restores a request body after the capturing handler has drained
// it, so the wrapped transport still has something to send.
type rewriteBody struct{ base http.RoundTripper }

func (w rewriteBody) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		r.ContentLength = int64(len(raw))
	}
	return w.base.RoundTrip(r)
}

// TestCreateCacheSendsTheTTLAndReturnsItsName confirms WithCacheTTL reaches
// the request and CreateCache returns the API's resource name.
func TestCreateCacheSendsTheTTLAndReturnsItsName(t *testing.T) {
	var body map[string]any
	c := cacheClient(t, &body, http.StatusOK,
		`{"name":"cachedContents/abc123"}`,
		WithCacheTTL(90*time.Second))

	name, err := c.CreateCache(context.Background(),
		[]message.Message{message.NewUserMessage("remember this")}, nil)
	if err != nil {
		t.Fatalf("CreateCache: %v", err)
	}

	if name != "cachedContents/abc123" {
		t.Errorf("name = %q, want the resource name the API returned", name)
	}
	if got := body["ttl"]; got != "90s" {
		t.Errorf(
			"ttl = %v, want %q -- WithCacheTTL never reached the request",
			got,
			"90s",
		)
	}
}

// TestCreateCacheWithNoTTLSendsNone confirms an unset TTL is omitted from the
// request rather than sent as a zero value.
func TestCreateCacheWithNoTTLSendsNone(t *testing.T) {
	var body map[string]any
	c := cacheClient(
		t,
		&body,
		http.StatusOK,
		`{"name":"cachedContents/abc123"}`,
	)

	if _, err := c.CreateCache(
		context.Background(),
		[]message.Message{
			message.NewUserMessage("remember this"),
		},
		nil,
	); err != nil {
		t.Fatalf("CreateCache: %v", err)
	}

	if got, present := body["ttl"]; present && got != "" {
		t.Errorf("ttl = %v, want it absent when the caller set none", got)
	}
}

// TestCreateCacheReportsAFailure confirms a failed create returns an error and
// an empty name, never an empty name alone.
func TestCreateCacheReportsAFailure(t *testing.T) {
	var body map[string]any
	c := cacheClient(t, &body, http.StatusBadRequest,
		`{"error":{"code":400,"message":"nope"}}`)

	name, err := c.CreateCache(context.Background(),
		[]message.Message{message.NewUserMessage("remember this")}, nil)
	if err == nil {
		t.Fatal("CreateCache returned no error on a 400")
	}
	if name != "" {
		t.Errorf("name = %q, want empty alongside the error", name)
	}
}
