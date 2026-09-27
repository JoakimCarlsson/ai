package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

// newResponsesServer returns a test server that replies with the given
// Responses-API JSON, ignoring the request body.
func newResponsesServer(
	t *testing.T,
	_ *map[string]any,
	response string,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, response)
		}))
}

// A Responses body whose prompt was mostly served from the cache. The API
// reports input_tokens as the WHOLE prompt and cached_tokens as the part of
// it that was a cache hit -- a subset, not an addition.
const responsesCachedBody = `{"id":"resp_c","object":"response","status":"completed",` +
	`"output":[{"type":"message","role":"assistant",` +
	`"content":[{"type":"output_text","text":"hi"}]}],` +
	`"usage":{"input_tokens":1000,"input_tokens_details":{"cached_tokens":800},` +
	`"output_tokens":50,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":1050}}`

// TestResponsesUsageExcludesCachedFromInput confirms the Responses mapping
// matches the chat-completions contract: InputTokens is the uncached input,
// and CacheReadTokens carries the rest. Covered on both the plain and
// streamed path.
func TestResponsesUsageExcludesCachedFromInput(t *testing.T) {
	check := func(t *testing.T, u llm.TokenUsage) {
		t.Helper()
		if u.InputTokens != 200 {
			t.Errorf(
				"InputTokens = %d, want 200 (1000 prompt minus 800 cached)",
				u.InputTokens,
			)
		}
		if u.CacheReadTokens != 800 {
			t.Errorf("CacheReadTokens = %d, want 800", u.CacheReadTokens)
		}
		if u.OutputTokens != 50 {
			t.Errorf("OutputTokens = %d, want 50", u.OutputTokens)
		}
		if got := u.InputTokens + u.CacheReadTokens + u.CacheCreationTokens; got != 1000 {
			t.Errorf(
				"input + cache read + cache write = %d, want the whole prompt, 1000",
				got,
			)
		}
	}

	t.Run("send", func(t *testing.T) {
		srv := newResponsesServer(t, nil, responsesCachedBody)
		defer srv.Close()
		client := NewResponsesLLM(
			WithResponsesAPIKey("test-key"),
			WithResponsesBaseURL(srv.URL),
			WithResponsesModel(llm.Model{APIModel: "gpt-4o-mini"}),
		)
		resp, err := client.SendMessages(context.Background(),
			[]message.Message{message.NewUserMessage("hi")}, nil)
		if err != nil {
			t.Fatalf("SendMessages: %v", err)
		}
		check(t, resp.Usage)
	})

	t.Run("stream", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, "event: response.completed\n"+
					`data: {"type":"response.completed","sequence_number":1,"response":`+
					responsesCachedBody+"}\n\n")
			}))
		defer srv.Close()
		client := NewResponsesLLM(
			WithResponsesAPIKey("test-key"),
			WithResponsesBaseURL(srv.URL),
			WithResponsesModel(llm.Model{APIModel: "gpt-4o-mini"}),
		)
		var final *llm.Response
		for ev := range client.StreamResponse(context.Background(),
			[]message.Message{message.NewUserMessage("hi")}, nil) {
			if ev.Type == types.EventError {
				t.Fatalf("stream error: %v", ev.Error)
			}
			if ev.Type == types.EventComplete {
				final = ev.Response
			}
		}
		if final == nil {
			t.Fatal("stream ended without a complete event")
		}
		check(t, final.Usage)
	})
}

// TestResponsesUsageClampsACachedFigureAboveTheTotal confirms a cached
// figure above the reported total clamps InputTokens to zero rather than
// going negative.
func TestResponsesUsageClampsACachedFigureAboveTheTotal(t *testing.T) {
	body := `{"id":"resp_x","object":"response","status":"completed",` +
		`"output":[{"type":"message","role":"assistant",` +
		`"content":[{"type":"output_text","text":"hi"}]}],` +
		`"usage":{"input_tokens":10,"input_tokens_details":{"cached_tokens":40},"output_tokens":1}}`
	srv := newResponsesServer(t, nil, body)
	defer srv.Close()
	client := NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-4o-mini"}),
	)
	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	if resp.Usage.InputTokens != 0 {
		t.Errorf(
			"InputTokens = %d, want 0 when cached exceeds the total",
			resp.Usage.InputTokens,
		)
	}
}
