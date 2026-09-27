package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

// A streamed chat completion, as the wire delivers it: content, then a finish,
// then a usage-only chunk. usageExtras is spliced into that last chunk's usage
// object and topExtras into the chunk itself, so a test can put a field where
// a provider would put it.
func sseChunks(usageExtras, topExtras string) []string {
	return []string{
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"m",` +
			`"choices":[{"index":0,"delta":{"role":"assistant","content":"hi"}}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"m",` +
			`"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"m"` +
			topExtras + `,"choices":[],"usage":{"prompt_tokens":3,` +
			`"completion_tokens":2,"total_tokens":5` + usageExtras + `}}`,
	}
}

func streamServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("test server cannot flush, so it cannot stream")
			}
			for _, c := range chunks {
				if _, err := w.Write(
					[]byte("data: " + c + "\n\n"),
				); err != nil {
					return
				}
				flusher.Flush()
			}
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		}))
}

// drainStream returns the response carried by the completion event, failing
// the test on an error event.
func drainStream(t *testing.T, events <-chan llm.Event) *llm.Response {
	t.Helper()
	var resp *llm.Response
	for ev := range events {
		switch {
		case ev.Type == types.EventError && ev.Error != nil:
			t.Fatalf("stream returned an error event: %v", ev.Error)
		case ev.Type == types.EventComplete && ev.Response != nil:
			resp = ev.Response
		}
	}
	if resp == nil {
		t.Fatal("stream ended with no completion event")
	}
	return resp
}

func streamingClient(t *testing.T, url string) llm.LLM {
	t.Helper()
	return NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(url),
		WithModel(llm.Model{APIModel: "m"}),
		WithResponseMetadataField("usage.cost", "usage.cost"),
		WithResponseMetadataField("cost", "toplevel.cost"),
	)
}

// TestStreamedResponseCarriesNestedUsageMetadata confirms a usage-nested
// metadata field reaches ProviderMetadata on the streaming path.
func TestStreamedResponseCarriesNestedUsageMetadata(t *testing.T) {
	srv := streamServer(t, sseChunks(`,"cost":0.00031`, ``))
	defer srv.Close()

	resp := drainStream(t, streamingClient(t, srv.URL).StreamResponse(
		context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil),
	)

	got, ok := resp.ProviderMetadata["usage.cost"]
	if !ok {
		t.Fatalf("usage.cost absent from streamed metadata %v -- the gateway "+
			"reported a cost and the ledger will fall back to table pricing",
			resp.ProviderMetadata)
	}
	if got != 0.00031 {
		t.Errorf("usage.cost = %v, want 0.00031", got)
	}
}

// TestStreamedResponseCarriesTopLevelMetadata confirms a top-level metadata
// field also reaches ProviderMetadata on the streaming path.
func TestStreamedResponseCarriesTopLevelMetadata(t *testing.T) {
	srv := streamServer(t, sseChunks(``, `,"cost":0.99`))
	defer srv.Close()

	resp := drainStream(t, streamingClient(t, srv.URL).StreamResponse(
		context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil),
	)

	got, ok := resp.ProviderMetadata["toplevel.cost"]
	if !ok {
		t.Fatalf(
			"top-level cost absent from streamed metadata %v",
			resp.ProviderMetadata,
		)
	}
	if got != 0.99 {
		t.Errorf("toplevel.cost = %v, want 0.99", got)
	}
}

// TestStreamedResponseWithNoCostReportsNothing confirms a configured field
// the provider never sends stays absent rather than erroring.
func TestStreamedResponseWithNoCostReportsNothing(t *testing.T) {
	srv := streamServer(t, sseChunks(``, ``))
	defer srv.Close()

	resp := drainStream(t, streamingClient(t, srv.URL).StreamResponse(
		context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil),
	)

	if v, ok := resp.ProviderMetadata["usage.cost"]; ok {
		t.Errorf("usage.cost = %v, want absent when the provider sent none", v)
	}
	if v, ok := resp.ProviderMetadata["toplevel.cost"]; ok {
		t.Errorf(
			"toplevel.cost = %v, want absent when the provider sent none",
			v,
		)
	}
}

// TestStreamedUsageCountsSurviveTheMetadataCollection confirms collecting
// extras off the chunks does not disturb the accumulator's own token counts
// and content.
func TestStreamedUsageCountsSurviveTheMetadataCollection(t *testing.T) {
	srv := streamServer(t, sseChunks(`,"cost":0.00031`, ``))
	defer srv.Close()

	resp := drainStream(t, streamingClient(t, srv.URL).StreamResponse(
		context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil),
	)

	if resp.Usage.InputTokens != 3 {
		t.Errorf("InputTokens = %d, want 3", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 2 {
		t.Errorf("OutputTokens = %d, want 2", resp.Usage.OutputTokens)
	}
	if resp.Content != "hi" {
		t.Errorf("Content = %q, want %q", resp.Content, "hi")
	}
}

// TestStreamedMetadataIsSkippedWhenNothingIsConfigured confirms
// ProviderMetadata is nil, not an empty map, when no field is configured.
func TestStreamedMetadataIsSkippedWhenNothingIsConfigured(t *testing.T) {
	srv := streamServer(t, sseChunks(`,"cost":0.00031`, ``))
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(llm.Model{APIModel: "m"}),
	)
	resp := drainStream(t, client.StreamResponse(
		context.Background(),
		[]message.Message{message.NewUserMessage("hi")},
		nil,
	))

	if resp.ProviderMetadata != nil {
		t.Errorf("ProviderMetadata = %v, want nil when no field was configured",
			resp.ProviderMetadata)
	}
}
