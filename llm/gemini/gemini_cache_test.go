package gemini

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"google.golang.org/genai"
)

// mustBuildConfig is buildConfig for tests that expect it to succeed.
func (c *Client) mustBuildConfig(
	t *testing.T,
	systemMessages []string,
	tools []tool.BaseTool,
) *genai.GenerateContentConfig {
	t.Helper()
	cfg, err := c.buildConfig(systemMessages, tools)
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	return cfg
}

// TestWithCachedContentSetsConfig verifies WithCachedContent populates
// config.CachedContent and leaves SystemInstruction and Tools unset.
func TestWithCachedContentSetsConfig(t *testing.T) {
	cfg := clientWith(WithCachedContent("cachedContents/x")).
		mustBuildConfig(t, nil, nil)

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

// TestWithCachedContentRejectsInlineContent verifies a request that passes
// system messages or tools alongside a cache fails instead of silently
// dropping them.
func TestWithCachedContentRejectsInlineContent(t *testing.T) {
	cases := map[string]struct {
		system []string
		tools  []tool.BaseTool
	}{
		"system messages": {system: []string{"you are a helpful assistant"}},
		"tools":           {tools: []tool.BaseTool{stubTool{name: "lookup"}}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := clientWith(WithCachedContent("cachedContents/x")).
				buildConfig(tc.system, tc.tools)
			if !errors.Is(err, ErrCachedContentConflict) {
				t.Errorf("err = %v, want ErrCachedContentConflict", err)
			}
		})
	}
}

// TestWithDisableCacheIgnoresCachedContent verifies WithDisableCache overrides
// a configured cache: CachedContent stays empty and the system instruction is
// sent inline.
func TestWithDisableCacheIgnoresCachedContent(t *testing.T) {
	cfg := clientWith(
		WithCachedContent("cachedContents/x"),
		WithDisableCache(),
	).mustBuildConfig(t, []string{"you are a helpful assistant"}, nil)

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
	cfg := clientWith().mustBuildConfig(t, nil, nil)
	if cfg.CachedContent != "" {
		t.Errorf("CachedContent = %q, want empty by default", cfg.CachedContent)
	}
}

// cacheClient builds a client through NewLLM whose requests are captured into
// body and answered with status and reply, and returns it as the
// [llm.ContextCacheProvider] the tracing wrapper must preserve.
func cacheClient(
	t *testing.T,
	body *map[string]any,
	status int,
	reply string,
	opts ...Option,
) llm.ContextCacheProvider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, reply)
		}))
	t.Cleanup(srv.Close)

	opts = append([]Option{
		WithAPIKey("test-key"),
		WithModel(llm.Model{APIModel: "gemini-2.0-flash"}),
		WithHTTPClient(&http.Client{
			Transport: newCapturingRT(srv.Listener.Addr().String(), body),
		}),
	}, opts...)
	ccp, ok := NewLLM(opts...).(llm.ContextCacheProvider)
	if !ok {
		t.Fatal("NewLLM result does not implement llm.ContextCacheProvider")
	}
	return ccp
}

// createCache asks ccp to cache a single user message and the given tools.
func createCache(
	ccp llm.ContextCacheProvider,
	tools ...tool.BaseTool,
) (string, error) {
	return ccp.CreateCache(
		context.Background(),
		[]message.Message{message.NewUserMessage("remember this")},
		tools,
	)
}

// TestCreateCacheSendsTheTTLAndReturnsItsName confirms WithCacheTTL reaches
// the request and CreateCache returns the API's resource name.
func TestCreateCacheSendsTheTTLAndReturnsItsName(t *testing.T) {
	var body map[string]any
	ccp := cacheClient(t, &body, http.StatusOK,
		`{"name":"cachedContents/abc123"}`,
		WithCacheTTL(90*time.Second))

	name, err := createCache(ccp)
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
	ccp := cacheClient(t, &body, http.StatusOK,
		`{"name":"cachedContents/abc123"}`)

	if _, err := createCache(ccp); err != nil {
		t.Fatalf("CreateCache: %v", err)
	}

	if got, present := body["ttl"]; present {
		t.Errorf("ttl = %v, want it absent when the caller set none", got)
	}
}

// TestCreateCacheCarriesToolChoice confirms WithToolChoice is stored in the
// cache, since a request that uses the cache cannot send a tool config.
func TestCreateCacheCarriesToolChoice(t *testing.T) {
	var body map[string]any
	ccp := cacheClient(t, &body, http.StatusOK,
		`{"name":"cachedContents/abc123"}`,
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}))

	if _, err := createCache(ccp, stubTool{name: "lookup"}); err != nil {
		t.Fatalf("CreateCache: %v", err)
	}

	toolConfig, _ := body["toolConfig"].(map[string]any)
	fc, _ := toolConfig["functionCallingConfig"].(map[string]any)
	if got := fc["mode"]; got != "ANY" {
		t.Errorf("toolConfig.functionCallingConfig.mode = %v, want ANY", got)
	}
}

// TestCreateCacheReportsAFailure confirms a failed create returns an error and
// an empty name, never an empty name alone.
func TestCreateCacheReportsAFailure(t *testing.T) {
	var body map[string]any
	ccp := cacheClient(t, &body, http.StatusBadRequest,
		`{"error":{"code":400,"message":"nope"}}`)

	name, err := createCache(ccp)
	if err == nil {
		t.Fatal("CreateCache returned no error on a 400")
	}
	if name != "" {
		t.Errorf("name = %q, want empty alongside the error", name)
	}
}
