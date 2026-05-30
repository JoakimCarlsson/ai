package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

// TestPreparedParamsStopSequencesArray verifies that all provided stop
// sequences are sent as an array (OfStringArray), not just the first one.
func TestPreparedParamsStopSequencesArray(t *testing.T) {
	c := &Client{options: Options{
		stopSequences: []string{"END", "STOP", "HALT"},
	}}

	params := c.preparedParams(nil, nil)

	if params.Stop.OfString.Valid() {
		t.Fatalf(
			"expected OfString to be unset, got %q",
			params.Stop.OfString.Value,
		)
	}
	got := params.Stop.OfStringArray
	want := []string{"END", "STOP", "HALT"}
	if len(got) != len(want) {
		t.Fatalf(
			"expected %d stop sequences, got %d: %v",
			len(want),
			len(got),
			got,
		)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stop[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestPreparedParamsStopSequencesCappedAtFour verifies that the OpenAI-native
// limit of 4 stop sequences is enforced.
func TestPreparedParamsStopSequencesCappedAtFour(t *testing.T) {
	c := &Client{options: Options{
		stopSequences: []string{"1", "2", "3", "4", "5", "6"},
	}}

	params := c.preparedParams(nil, nil)

	if len(params.Stop.OfStringArray) != 4 {
		t.Fatalf("expected stop sequences capped at 4, got %d: %v",
			len(params.Stop.OfStringArray), params.Stop.OfStringArray)
	}
}

// TestRequestOptionsTopK verifies that top_k yields a request option only on
// the compatible-provider path: it requires both WithTopK and a custom base
// URL, since OpenAI/Azure proper reject top_k.
func TestRequestOptionsTopK(t *testing.T) {
	k := int64(40)

	none := (&Client{options: Options{baseURL: "https://example.test"}}).requestOptions()
	if len(none) != 0 {
		t.Errorf(
			"expected no request options when top_k unset, got %d",
			len(none),
		)
	}

	native := (&Client{options: Options{topK: &k}}).requestOptions()
	if len(native) != 0 {
		t.Errorf("expected no top_k injection without a custom base URL "+
			"(OpenAI/Azure reject it), got %d", len(native))
	}

	compat := (&Client{options: Options{topK: &k, baseURL: "https://example.test"}}).requestOptions()
	if len(compat) != 1 {
		t.Fatalf(
			"expected one request option on the compatible path, got %d",
			len(compat),
		)
	}
}

// TestWireTopKAndStop confirms that top_k reaches the request body and that all
// stop sequences are serialized as a JSON array on the wire.
func TestWireTopKAndStop(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"x","object":"chat.completion",`+
				`"choices":[{"index":0,"message":{"role":"assistant",`+
				`"content":"hi"},"finish_reason":"stop"}],`+
				`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
		}))
	defer srv.Close()

	llm := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithTopK(40),
		WithStopSequences("END", "STOP", "HALT"),
	)

	_, err := llm.SendMessages(
		context.Background(),
		[]message.Message{message.NewUserMessage("hello")},
		nil,
	)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if got, ok := body["top_k"].(float64); !ok || int64(got) != 40 {
		t.Errorf("expected top_k=40 in request body, got %v (%T)",
			body["top_k"], body["top_k"])
	}

	stop, ok := body["stop"].([]any)
	if !ok {
		t.Fatalf("expected stop to be a JSON array, got %v (%T)",
			body["stop"], body["stop"])
	}
	want := []string{"END", "STOP", "HALT"}
	if len(stop) != len(want) {
		t.Fatalf("expected %d stop sequences on the wire, got %d: %v",
			len(want), len(stop), stop)
	}
	for i, v := range want {
		if stop[i] != v {
			t.Errorf("wire stop[%d] = %v, want %q", i, stop[i], v)
		}
	}
}

// TestWireRequestJSONField confirms WithRequestJSONField injects arbitrary
// top-level fields (objects and arrays) into the request body.
func TestWireRequestJSONField(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(
		t,
		&body,
		`{"id":"x","object":"chat.completion",`+
			`"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},`+
			`"finish_reason":"stop"}],`+
			`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`,
	)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "x"}),
		WithRequestJSONField(
			"provider",
			map[string]any{"allow_fallbacks": false},
		),
		WithRequestJSONField("models", []string{"a", "b"}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	provider, ok := body["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider object, got %v (%T)",
			body["provider"], body["provider"])
	}
	if provider["allow_fallbacks"] != false {
		t.Errorf("provider.allow_fallbacks = %v, want false",
			provider["allow_fallbacks"])
	}
	models, ok := body["models"].([]any)
	if !ok || len(models) != 2 || models[0] != "a" || models[1] != "b" {
		t.Errorf("models = %v, want [a b]", body["models"])
	}
}

// TestResponseMetadataField confirms WithResponseMetadataField surfaces a
// top-level response field into ProviderMetadata under the namespaced key.
func TestResponseMetadataField(t *testing.T) {
	srv := newCompletionServer(t, nil, `{"id":"x","object":"chat.completion",`+
		`"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},`+
		`"finish_reason":"stop"}],`+
		`"citations":["https://a.test","https://b.test"],`+
		`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "x"}),
		WithResponseMetadataField("citations", "perplexity.citations"),
	)

	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	cites, ok := resp.ProviderMetadata["perplexity.citations"].([]any)
	if !ok || len(cites) != 2 || cites[0] != "https://a.test" {
		t.Fatalf("expected citations in metadata, got %v",
			resp.ProviderMetadata)
	}
}

// TestUsageReasoningAndDeepSeekCache verifies that reasoning_tokens and
// DeepSeek's top-level prompt_cache_hit_tokens (an SDK extra field) are mapped.
func TestUsageReasoningAndDeepSeekCache(t *testing.T) {
	srv := newCompletionServer(t, nil, `{"id":"x","object":"chat.completion",`+
		`"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},`+
		`"finish_reason":"stop"}],`+
		`"usage":{"prompt_tokens":100,"completion_tokens":40,"total_tokens":140,`+
		`"prompt_cache_hit_tokens":80,"prompt_cache_miss_tokens":20,`+
		`"completion_tokens_details":{"reasoning_tokens":12}}}`)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "deepseek-chat"}),
	)

	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if resp.Usage.CacheReadTokens != 80 {
		t.Errorf("CacheReadTokens = %d, want 80", resp.Usage.CacheReadTokens)
	}
	if resp.Usage.InputTokens != 20 {
		t.Errorf("InputTokens = %d, want 20 (prompt - cache hit)",
			resp.Usage.InputTokens)
	}
	if resp.Usage.ReasoningTokens != 12 {
		t.Errorf("ReasoningTokens = %d, want 12", resp.Usage.ReasoningTokens)
	}
}

// newCompletionServer returns a test server that captures the request body into
// capture (when non-nil) and replies with the given chat-completion JSON.
func newCompletionServer(
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
