package stt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

type fakeStreamingClient struct {
	emit []StreamResult
	err  error
}

func (f *fakeStreamingClient) transcribe(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return &Response{}, nil
}

func (f *fakeStreamingClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return &Response{}, nil
}

func (f *fakeStreamingClient) streamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	_ ...Option,
) (<-chan StreamResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(chan StreamResult, len(f.emit))
	go func() {
		defer close(out)
		for _, r := range f.emit {
			select {
			case <-ctx.Done():
				return
			case out <- r:
			}
		}
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-audio:
				if !ok {
					return
				}
			}
		}
	}()
	return out, nil
}

type fakeBatchClient struct{}

func (fakeBatchClient) transcribe(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return &Response{}, nil
}

func (fakeBatchClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return &Response{}, nil
}

func newTestSTT(client SpeechToTextClient) SpeechToText {
	return &baseSpeechToText[SpeechToTextClient]{
		options:   transcriptionClientOptions{model: model.TranscriptionModel{Provider: "fake"}},
		client:    client,
		streaming: resolveStreaming(client),
	}
}

func TestStreamTranscribeForwardsResults(t *testing.T) {
	results := []StreamResult{
		{Text: "hello", IsFinal: false},
		{Text: "hello world", IsFinal: true},
	}
	stt := newTestSTT(&fakeStreamingClient{emit: results})
	if !stt.SupportsStreaming() {
		t.Fatal("SupportsStreaming should be true")
	}

	audio := make(chan []byte)
	close(audio)

	out, err := stt.StreamTranscribe(context.Background(), audio)
	if err != nil {
		t.Fatalf("StreamTranscribe: %v", err)
	}
	got := drain(out)
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].Text != "hello" || got[0].IsFinal {
		t.Errorf("unexpected first result: %+v", got[0])
	}
	if got[1].Text != "hello world" || !got[1].IsFinal {
		t.Errorf("unexpected second result: %+v", got[1])
	}
}

func TestStreamTranscribeUnsupported(t *testing.T) {
	stt := newTestSTT(fakeBatchClient{})
	if stt.SupportsStreaming() {
		t.Fatal("SupportsStreaming should be false")
	}
	audio := make(chan []byte)
	close(audio)
	_, err := stt.StreamTranscribe(context.Background(), audio)
	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("expected ErrStreamingNotSupported, got %v", err)
	}
}

func TestStreamTranscribeSetupError(t *testing.T) {
	want := errors.New("dial failed")
	stt := newTestSTT(&fakeStreamingClient{err: want})
	audio := make(chan []byte)
	close(audio)
	_, err := stt.StreamTranscribe(context.Background(), audio)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestStreamTranscribeCtxCancel(t *testing.T) {
	stt := newTestSTT(&fakeStreamingClient{emit: []StreamResult{{Text: "one"}}})
	audio := make(chan []byte)
	ctx, cancel := context.WithCancel(context.Background())
	out, err := stt.StreamTranscribe(ctx, audio)
	if err != nil {
		t.Fatalf("StreamTranscribe: %v", err)
	}
	first, ok := <-out
	if !ok {
		t.Fatal("channel closed before first result")
	}
	if first.Text != "one" {
		t.Fatalf("unexpected first: %+v", first)
	}
	cancel()
	select {
	case _, more := <-out:
		if more {
			drain(out)
		}
	case <-time.After(time.Second):
		t.Fatal("channel did not close within 1s of ctx cancel")
	}
}

func drain(ch <-chan StreamResult) []StreamResult {
	var out []StreamResult
	for r := range ch {
		out = append(out, r)
	}
	return out
}
