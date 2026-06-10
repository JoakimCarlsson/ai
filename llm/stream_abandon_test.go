package llm

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// stubStreamLLM emulates a vendor client whose stream goroutine emits a
// scripted event sequence with bare (blocking) sends — the same shape as the
// real vendor producers — then closes the channel.
type stubStreamLLM struct {
	events []Event
}

func (s *stubStreamLLM) SendMessages(
	context.Context, []message.Message, []tool.BaseTool,
) (*Response, error) {
	return nil, errors.New("not implemented")
}

func (s *stubStreamLLM) SendMessagesWithStructuredOutput(
	context.Context, []message.Message, []tool.BaseTool, *schema.StructuredOutputInfo,
) (*Response, error) {
	return nil, errors.New("not implemented")
}

func (s *stubStreamLLM) stream() <-chan Event {
	ch := make(chan Event)
	go func() {
		defer close(ch)
		for _, evt := range s.events {
			ch <- evt
		}
	}()
	return ch
}

func (s *stubStreamLLM) StreamResponse(
	context.Context, []message.Message, []tool.BaseTool,
) <-chan Event {
	return s.stream()
}

func (s *stubStreamLLM) StreamResponseWithStructuredOutput(
	context.Context, []message.Message, []tool.BaseTool, *schema.StructuredOutputInfo,
) <-chan Event {
	return s.stream()
}

func (s *stubStreamLLM) Model() model.Model             { return model.Model{} }
func (s *stubStreamLLM) SupportsStructuredOutput() bool { return true }

// twoErrorEvents is the provider+retry double-emission shape that strands the
// forwarder when the consumer stops after the first error event.
func twoErrorEvents() []Event {
	streamErr := errors.New("stream failed")
	return []Event{
		{Type: types.EventError, Error: streamErr},
		{Type: types.EventError, Error: streamErr},
	}
}

// abandonAfterFirstError reads one event, asserts it is an error, then cancels
// ctx and never reads the channel again — the consumer-abandonment pattern.
func abandonAfterFirstError(
	t *testing.T,
	ch <-chan Event,
	cancel context.CancelFunc,
) {
	t.Helper()
	evt := <-ch
	if evt.Type != types.EventError {
		t.Fatalf("first event type = %v, want %v", evt.Type, types.EventError)
	}
	cancel()
}

// waitForGoroutineBaseline polls until the goroutine count drops back to the
// pre-stream baseline, failing with a full stack dump if the forwarder (or the
// stub producer) is still alive after the deadline.
func waitForGoroutineBaseline(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseline {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	t.Fatalf("goroutines stuck above baseline %d (now %d):\n%s",
		baseline, runtime.NumGoroutine(), buf[:n])
}

// TestStreamResponseAbandonedConsumerReleasesForwarder reproduces the strand:
// the inner stream emits two error events, the consumer reads only the first
// and cancels ctx. The forwarding goroutine must drain the inner channel and
// exit (ending its tracing span) — before the fix it blocked forever on the
// second send to the abandoned channel.
func TestStreamResponseAbandonedConsumerReleasesForwarder(t *testing.T) {
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inner := &stubStreamLLM{events: twoErrorEvents()}
	ch := WithTracing(inner, TracingAttrs{}).StreamResponse(ctx, nil, nil)

	abandonAfterFirstError(t, ch, cancel)
	waitForGoroutineBaseline(t, baseline)
}

// TestStreamStructuredOutputAbandonedConsumerReleasesForwarder covers the
// structured-output forwarder variant with the same abandonment pattern.
func TestStreamStructuredOutputAbandonedConsumerReleasesForwarder(t *testing.T) {
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inner := &stubStreamLLM{events: twoErrorEvents()}
	ch := WithTracing(inner, TracingAttrs{}).
		StreamResponseWithStructuredOutput(ctx, nil, nil, nil)

	abandonAfterFirstError(t, ch, cancel)
	waitForGoroutineBaseline(t, baseline)
}
