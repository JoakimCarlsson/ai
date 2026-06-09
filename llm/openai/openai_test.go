package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
)

// stubTool is a no-op BaseTool used to populate the request's tools slice so
// tool-choice assertions have something to attach to.
type stubTool struct{ name string }

func (s stubTool) Info() tool.Info {
	return tool.Info{
		Name:        s.name,
		Description: "d",
		Parameters:  map[string]any{},
	}
}

func (s stubTool) Run(context.Context, tool.Call) (tool.Response, error) {
	return tool.Response{}, nil
}

const completionOK = `{"id":"x","object":"chat.completion",` +
	`"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},` +
	`"finish_reason":"stop"}],` +
	`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`

// TestWireToolChoiceRequired confirms a Required choice serializes to the
// "required" string form when tools are present.
func TestWireToolChoiceRequired(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")},
		[]tool.BaseTool{stubTool{name: "get_weather"}}); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if got, _ := body["tool_choice"].(string); got != "required" {
		t.Errorf("tool_choice = %v, want %q", body["tool_choice"], "required")
	}
}

// TestWireToolChoiceSpecific confirms a Specific choice serializes to the
// named-function object form naming the tool.
func TestWireToolChoiceSpecific(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithToolChoice(llm.ToolChoice{
			Mode: llm.ToolChoiceSpecific,
			Name: "get_weather",
		}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")},
		[]tool.BaseTool{stubTool{name: "get_weather"}}); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	tc, ok := body["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %v (%T), want object",
			body["tool_choice"], body["tool_choice"])
	}
	if tc["type"] != "function" {
		t.Errorf("tool_choice.type = %v, want function", tc["type"])
	}
	fn, ok := tc["function"].(map[string]any)
	if !ok || fn["name"] != "get_weather" {
		t.Errorf(
			"tool_choice.function = %v, want name=get_weather",
			tc["function"],
		)
	}
}

// TestWireToolChoiceOmittedWithoutTools confirms no tool_choice field is emitted
// when the tools slice is empty.
func TestWireToolChoiceOmittedWithoutTools(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if _, present := body["tool_choice"]; present {
		t.Errorf("tool_choice should be omitted with no tools, got %v",
			body["tool_choice"])
	}
}

// TestToolChoiceSpecificEmptyNameRejected confirms a Specific choice with no
// name is rejected before any request is sent.
func TestToolChoiceSpecificEmptyNameRejected(t *testing.T) {
	client := NewLLM(
		WithAPIKey("test-key"),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceSpecific}),
	)

	_, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")},
		[]tool.BaseTool{stubTool{name: "get_weather"}})
	if !errors.Is(err, llm.ErrToolChoiceNameRequired) {
		t.Fatalf("expected ErrToolChoiceNameRequired, got %v", err)
	}
}

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

// TestResponseRequestIDAndHeaders confirms the provider request id and the
// allowlisted rate-limit header are surfaced on the response, while a
// non-allowlisted header (here, an auth echo) is dropped.
func TestResponseRequestIDAndHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("x-request-id", "req_test_123")
			w.Header().Set("x-ratelimit-remaining-requests", "42")
			w.Header().Set("authorization", "Bearer leak")
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, completionOK)
		}))
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if resp.RequestID != "req_test_123" {
		t.Errorf("RequestID = %q, want %q", resp.RequestID, "req_test_123")
	}
	if got := resp.ResponseHeaders.Get(
		"x-ratelimit-remaining-requests",
	); got != "42" {
		t.Errorf("x-ratelimit-remaining-requests = %q, want %q", got, "42")
	}
	if got := resp.ResponseHeaders.Get("x-request-id"); got != "req_test_123" {
		t.Errorf("x-request-id header = %q, want %q", got, "req_test_123")
	}
	if got := resp.ResponseHeaders.Get("authorization"); got != "" {
		t.Errorf("authorization should not be retained, got %q", got)
	}
}

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

// TestWithHTTPClientTransportUsed confirms a client injected via WithHTTPClient
// handles outgoing requests: the wrapped transport's counter increments, proving
// the SDK default client was replaced.
func TestWithHTTPClientTransportUsed(t *testing.T) {
	srv := newCompletionServer(t, nil, completionOK)
	defer srv.Close()

	var n int
	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithHTTPClient(&http.Client{
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

// TestWireLogitBias confirms WithLogitBias emits logit_bias as a token-id ->
// bias object, and that it is omitted when the option is unset.
func TestWireLogitBias(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithLogitBias(map[string]int{"50256": -100, "1212": 5}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	bias, ok := body["logit_bias"].(map[string]any)
	if !ok {
		t.Fatalf("logit_bias = %v (%T), want object",
			body["logit_bias"], body["logit_bias"])
	}
	if got, _ := bias["50256"].(float64); int(got) != -100 {
		t.Errorf("logit_bias[50256] = %v, want -100", bias["50256"])
	}
	if got, _ := bias["1212"].(float64); int(got) != 5 {
		t.Errorf("logit_bias[1212] = %v, want 5", bias["1212"])
	}
}

// TestWireLogitBiasOmitted confirms logit_bias is absent when WithLogitBias is
// not set.
func TestWireLogitBiasOmitted(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if _, present := body["logit_bias"]; present {
		t.Errorf("logit_bias should be omitted when unset, got %v",
			body["logit_bias"])
	}
}

// TestWireLogprobs confirms WithLogprobs sets logprobs:true and top_logprobs:N
// on the request body, and that both are omitted when the option is unset.
func TestWireLogprobs(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithLogprobs(5),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if got, _ := body["logprobs"].(bool); !got {
		t.Errorf("logprobs = %v, want true", body["logprobs"])
	}
	if got, _ := body["top_logprobs"].(float64); int(got) != 5 {
		t.Errorf("top_logprobs = %v, want 5", body["top_logprobs"])
	}
}

// TestWireLogprobsOmitted confirms logprobs/top_logprobs are absent when
// WithLogprobs is not set.
func TestWireLogprobsOmitted(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if _, present := body["logprobs"]; present {
		t.Errorf("logprobs should be omitted when unset, got %v",
			body["logprobs"])
	}
	if _, present := body["top_logprobs"]; present {
		t.Errorf("top_logprobs should be omitted when unset, got %v",
			body["top_logprobs"])
	}
}

// TestResponseLogProbs confirms a completion carrying a logprobs.content block
// is mapped onto Response.LogProbs, including the top_logprobs alternatives.
func TestResponseLogProbs(t *testing.T) {
	const resp = `{"id":"x","object":"chat.completion","choices":[{"index":0,` +
		`"message":{"role":"assistant","content":"hi"},"finish_reason":"stop",` +
		`"logprobs":{"content":[` +
		`{"token":"hi","logprob":-0.5,"top_logprobs":[` +
		`{"token":"hi","logprob":-0.5},{"token":"yo","logprob":-1.2}]},` +
		`{"token":"!","logprob":-0.1,"top_logprobs":[]}]}}],` +
		`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`

	srv := newCompletionServer(t, nil, resp)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithLogprobs(2),
	)

	out, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if len(out.LogProbs) != 2 {
		t.Fatalf("len(LogProbs) = %d, want 2", len(out.LogProbs))
	}
	first := out.LogProbs[0]
	if first.Token != "hi" || first.LogProb != -0.5 {
		t.Errorf("LogProbs[0] = %+v, want token=hi logprob=-0.5", first)
	}
	if len(first.Top) != 2 {
		t.Fatalf("len(LogProbs[0].Top) = %d, want 2", len(first.Top))
	}
	if first.Top[1].Token != "yo" || first.Top[1].LogProb != -1.2 {
		t.Errorf("LogProbs[0].Top[1] = %+v, want token=yo logprob=-1.2",
			first.Top[1])
	}
}

// TestResponseLogProbsNilWhenAbsent confirms Response.LogProbs is nil when the
// completion carries no logprobs block.
func TestResponseLogProbsNilWhenAbsent(t *testing.T) {
	srv := newCompletionServer(t, nil, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	out, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if out.LogProbs != nil {
		t.Errorf("LogProbs = %v, want nil", out.LogProbs)
	}
}

// TestWireN confirms WithN emits n on the request body, and that it is omitted
// when the option is unset.
func TestWireN(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithN(3),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if got, _ := body["n"].(float64); int(got) != 3 {
		t.Errorf("n = %v, want 3", body["n"])
	}
}

// TestWireNOmitted confirms n is absent when WithN is not set.
func TestWireNOmitted(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if _, present := body["n"]; present {
		t.Errorf("n should be omitted when unset, got %v", body["n"])
	}
}

// TestResponseChoices confirms a completion with multiple choices populates
// Response.Choices, with the top-level fields mirroring choice 0.
func TestResponseChoices(t *testing.T) {
	const resp = `{"id":"x","object":"chat.completion","choices":[` +
		`{"index":0,"message":{"role":"assistant","content":"first"},` +
		`"finish_reason":"stop"},` +
		`{"index":1,"message":{"role":"assistant","content":"second"},` +
		`"finish_reason":"length"},` +
		`{"index":2,"message":{"role":"assistant","content":"third"},` +
		`"finish_reason":"stop"}],` +
		`"usage":{"prompt_tokens":1,"completion_tokens":3,"total_tokens":4}}`

	srv := newCompletionServer(t, nil, resp)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithN(3),
	)

	out, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if len(out.Choices) != 3 {
		t.Fatalf("len(Choices) = %d, want 3", len(out.Choices))
	}
	if out.Content != out.Choices[0].Content || out.Content != "first" {
		t.Errorf("Content = %q, want choice 0 %q", out.Content,
			out.Choices[0].Content)
	}
	if out.FinishReason != out.Choices[0].FinishReason {
		t.Errorf("FinishReason = %q, want choice 0 %q", out.FinishReason,
			out.Choices[0].FinishReason)
	}
	if out.Choices[1].Content != "second" {
		t.Errorf("Choices[1].Content = %q, want second", out.Choices[1].Content)
	}
	if out.Choices[1].FinishReason != message.FinishReasonMaxTokens {
		t.Errorf("Choices[1].FinishReason = %q, want max-tokens",
			out.Choices[1].FinishReason)
	}
}

// TestResponseChoicesEmptyForSingleChoice confirms Response.Choices stays empty
// for a single-choice completion so callers use the top-level fields.
func TestResponseChoicesEmptyForSingleChoice(t *testing.T) {
	srv := newCompletionServer(t, nil, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	out, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if len(out.Choices) != 0 {
		t.Errorf("Choices = %v, want empty for a single choice", out.Choices)
	}
	if out.Content != "hi" {
		t.Errorf("Content = %q, want hi", out.Content)
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
