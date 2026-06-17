package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/types"
)

// TestStreamNoChoicesEmitsSingleError locks the error-emission ownership:
// runStream returns the "no response choices" error WITHOUT emitting it, and
// ExecuteStreamWithRetry emits it exactly once. The pre-fix double emission
// (provider emit + retry-layer emit) stranded consumers that stopped reading
// after the first error event.
func TestStreamNoChoicesEmitsSingleError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(
				w,
				"data: {\"id\":\"x\",\"object\":\"chat.completion.chunk\",\"choices\":[]}\n\n",
			)
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	errCount := 0
	for evt := range client.StreamResponse(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil) {
		if evt.Type == types.EventError {
			errCount++
		}
	}
	if errCount != 1 {
		t.Fatalf("error events = %d, want exactly 1", errCount)
	}
}
