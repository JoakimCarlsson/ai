package fim_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/fim"
)

// decodeText treats each SSE data line as {"text":..,"done":..} and reports a
// finish reason on the final chunk. It mirrors what a vendor decode callback does.
func decodeText(data []byte) (fim.StreamChunk, bool) {
	var d struct {
		Text string `json:"text"`
		Done bool   `json:"done"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return fim.StreamChunk{}, false
	}
	chunk := fim.StreamChunk{Delta: d.Text}
	if d.Done {
		fr := fim.FinishReasonStop
		chunk.FinishReason = &fr
		chunk.Usage = &fim.Usage{InputTokens: 3, OutputTokens: 5}
	}
	return chunk, true
}

func collect(body string) []fim.Event {
	out := make(chan fim.Event)
	go func() {
		fim.StreamSSE(strings.NewReader(body), decodeText, out)
		close(out)
	}()
	var events []fim.Event
	for e := range out {
		events = append(events, e)
	}
	return events
}

func TestStreamSSE_AccumulatesAndCompletes(t *testing.T) {
	body := "data: {\"text\":\"Hello\"}\n" +
		"garbage line without prefix\n" +
		"\n" +
		"data: {\"text\":\" world\",\"done\":true}\n" +
		"data: [DONE]\n"

	events := collect(body)

	var deltas []string
	var complete *fim.Response
	for _, e := range events {
		switch e.Type {
		case fim.EventContentDelta:
			deltas = append(deltas, e.Content)
		case fim.EventComplete:
			complete = e.Response
		case fim.EventError:
			t.Fatalf("unexpected error event: %v", e.Error)
		}
	}

	if got := strings.Join(deltas, ""); got != "Hello world" {
		t.Errorf("delta content = %q, want %q", got, "Hello world")
	}
	if complete == nil {
		t.Fatal("expected a complete event")
	}
	if complete.Content != "Hello world" {
		t.Errorf("complete content = %q, want %q", complete.Content, "Hello world")
	}
	if complete.FinishReason != fim.FinishReasonStop {
		t.Errorf("finish reason = %q, want %q", complete.FinishReason, fim.FinishReasonStop)
	}
	if complete.Usage.OutputTokens != 5 {
		t.Errorf("output tokens = %d, want 5", complete.Usage.OutputTokens)
	}
}

// EOF without an explicit [DONE] must still emit a complete event.
func TestStreamSSE_CompletesOnEOF(t *testing.T) {
	events := collect("data: {\"text\":\"partial\"}\n")
	if len(events) == 0 || events[len(events)-1].Type != fim.EventComplete {
		t.Fatalf("expected trailing complete event, got %+v", events)
	}
	if events[len(events)-1].Response.Content != "partial" {
		t.Errorf("content = %q, want %q", events[len(events)-1].Response.Content, "partial")
	}
}
