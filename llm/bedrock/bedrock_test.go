package bedrock

import (
	"net/http"
	"testing"
)

func applyOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// TestCachingEnabledByDefault verifies that prompt caching is no longer forced
// off: a freshly configured client leaves disableCache false so cache_control
// breakpoints reach Bedrock.
func TestCachingEnabledByDefault(t *testing.T) {
	if applyOptions().disableCache {
		t.Fatal("expected caching enabled by default (disableCache=false)")
	}
}

// TestWithDisableCacheRespected verifies that callers can still opt out.
func TestWithDisableCacheRespected(t *testing.T) {
	if !applyOptions(WithDisableCache()).disableCache {
		t.Fatal("expected WithDisableCache() to set disableCache=true")
	}
}

// TestWithHTTPClientStored verifies WithHTTPClient records the client so it can
// be passed through to the underlying Anthropic-on-Bedrock child. A full
// transport round-trip is exercised in the anthropic package; here the Bedrock
// path additionally requires AWS SigV4 signing, so this asserts the wiring at
// the option level to match the existing option-test convention.
func TestWithHTTPClientStored(t *testing.T) {
	c := &http.Client{}
	if got := applyOptions(WithHTTPClient(c)).httpClient; got != c {
		t.Fatalf("WithHTTPClient did not store the client: got %v", got)
	}
	if applyOptions().httpClient != nil {
		t.Fatal("httpClient should be nil when WithHTTPClient is not used")
	}
}
