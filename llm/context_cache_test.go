package llm

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// stubCachingLLM is a vendor client that also implements
// [ContextCacheProvider], recording the messages it was asked to cache.
type stubCachingLLM struct {
	stubStreamLLM
	cached []message.Message
}

// CreateCache records the messages and returns a fixed resource name.
func (s *stubCachingLLM) CreateCache(
	_ context.Context,
	messages []message.Message,
	_ []tool.BaseTool,
) (string, error) {
	s.cached = messages
	return "cachedContents/stub", nil
}

// TestWithTracingPreservesContextCacheProvider confirms the tracing wrapper
// keeps [ContextCacheProvider] reachable and forwards the call.
func TestWithTracingPreservesContextCacheProvider(t *testing.T) {
	inner := &stubCachingLLM{}
	ccp, ok := WithTracing(inner, TracingAttrs{}).(ContextCacheProvider)
	if !ok {
		t.Fatal("wrapper does not implement ContextCacheProvider")
	}

	name, err := ccp.CreateCache(
		context.Background(),
		[]message.Message{message.NewUserMessage("remember this")},
		nil,
	)
	if err != nil {
		t.Fatalf("CreateCache: %v", err)
	}
	if name != "cachedContents/stub" {
		t.Errorf("name = %q, want the inner client's resource name", name)
	}
	if len(inner.cached) != 1 {
		t.Errorf("inner received %d messages, want 1", len(inner.cached))
	}
}

// TestWithTracingHidesContextCacheProviderWhenUnsupported confirms the
// wrapper does not claim [ContextCacheProvider] for a client without it.
func TestWithTracingHidesContextCacheProviderWhenUnsupported(t *testing.T) {
	wrapped := WithTracing(&stubStreamLLM{}, TracingAttrs{})
	if _, ok := wrapped.(ContextCacheProvider); ok {
		t.Error("wrapper claims ContextCacheProvider the inner lacks")
	}
}
