package xai

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

const responsesCachedBody = `{"id":"resp_c","object":"response",` +
	`"status":"completed","output":[{"type":"message","role":"assistant",` +
	`"content":[{"type":"output_text","text":"hi"}]}],` +
	`"usage":{"input_tokens":1000,` +
	`"input_tokens_details":{"cached_tokens":800},"output_tokens":50,` +
	`"output_tokens_details":{"reasoning_tokens":0},"total_tokens":1050}}`

// newUsageServer returns a test server that replies to every request with
// the given body under the given content type.
func newUsageServer(
	t *testing.T,
	contentType string,
	body string,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			_, _ = io.WriteString(w, body)
		}))
}

// newUsageClient returns a Responses client pointed at the given base URL.
func newUsageClient(baseURL string) llm.LLM {
	return NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(baseURL),
		WithResponsesModel(llm.Model{APIModel: "grok-4"}),
	)
}

// checkCachedUsage asserts the usage of responsesCachedBody splits the
// 1000-token prompt into 200 uncached and 800 cached tokens.
func checkCachedUsage(t *testing.T, u llm.TokenUsage) {
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
	total := u.InputTokens + u.CacheReadTokens + u.CacheCreationTokens
	if total != 1000 {
		t.Errorf("input + cache read + cache write = %d, want 1000", total)
	}
}

// TestResponsesUsageExcludesCachedFromInput confirms InputTokens is the
// uncached input and CacheReadTokens carries the rest, on both the plain
// and streamed path.
func TestResponsesUsageExcludesCachedFromInput(t *testing.T) {
	msgs := []message.Message{message.NewUserMessage("hi")}

	t.Run("send", func(t *testing.T) {
		srv := newUsageServer(t, "application/json", responsesCachedBody)
		defer srv.Close()
		resp, err := newUsageClient(srv.URL).
			SendMessages(context.Background(), msgs, nil)
		if err != nil {
			t.Fatalf("SendMessages: %v", err)
		}
		checkCachedUsage(t, resp.Usage)
	})

	t.Run("stream", func(t *testing.T) {
		body := "event: response.completed\n" +
			`data: {"type":"response.completed","sequence_number":1,` +
			`"response":` + responsesCachedBody + "}\n\n"
		srv := newUsageServer(t, "text/event-stream", body)
		defer srv.Close()
		var final *llm.Response
		for ev := range newUsageClient(srv.URL).
			StreamResponse(context.Background(), msgs, nil) {
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
		checkCachedUsage(t, final.Usage)
	})
}

// TestResponsesUsageClampsACachedFigureAboveTheTotal confirms a cached
// figure above the reported total clamps InputTokens to zero rather than
// going negative.
func TestResponsesUsageClampsACachedFigureAboveTheTotal(t *testing.T) {
	body := `{"id":"resp_x","object":"response","status":"completed",` +
		`"output":[{"type":"message","role":"assistant",` +
		`"content":[{"type":"output_text","text":"hi"}]}],` +
		`"usage":{"input_tokens":10,` +
		`"input_tokens_details":{"cached_tokens":40},"output_tokens":1}}`
	srv := newUsageServer(t, "application/json", body)
	defer srv.Close()
	resp, err := newUsageClient(srv.URL).SendMessages(context.Background(),
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
