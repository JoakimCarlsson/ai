package bedrock

import "testing"

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
