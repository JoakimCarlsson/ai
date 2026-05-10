package voice

import (
	"context"
	"errors"
	"io"
	"sync"

	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/tts"
	"github.com/joakimcarlsson/ai/types"
)

type fakeLLM struct {
	mu       sync.Mutex
	calls    int
	scripts  []func(ctx context.Context) <-chan llm.Event
	lastMsgs []message.Message
}

func (f *fakeLLM) push(script func(ctx context.Context) <-chan llm.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts = append(f.scripts, script)
}

func (f *fakeLLM) lastMessages() []message.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]message.Message, len(f.lastMsgs))
	copy(out, f.lastMsgs)
	return out
}

func (f *fakeLLM) StreamResponse(
	ctx context.Context,
	msgs []message.Message,
	_ []tool.BaseTool,
) <-chan llm.Event {
	f.mu.Lock()
	f.lastMsgs = make([]message.Message, len(msgs))
	copy(f.lastMsgs, msgs)
	if len(f.scripts) == 0 {
		f.mu.Unlock()
		ch := make(chan llm.Event)
		close(ch)
		return ch
	}
	script := f.scripts[0]
	f.scripts = f.scripts[1:]
	f.calls++
	f.mu.Unlock()
	return script(ctx)
}

func (f *fakeLLM) SendMessages(
	context.Context,
	[]message.Message,
	[]tool.BaseTool,
) (*llm.Response, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeLLM) SendMessagesWithStructuredOutput(
	context.Context,
	[]message.Message,
	[]tool.BaseTool,
	*schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeLLM) StreamResponseWithStructuredOutput(
	context.Context,
	[]message.Message,
	[]tool.BaseTool,
	*schema.StructuredOutputInfo,
) <-chan llm.Event {
	return nil
}

func (f *fakeLLM) Model() model.Model             { return model.Model{} }
func (f *fakeLLM) SupportsStructuredOutput() bool { return false }

type fakeSTT struct {
	results chan stt.StreamResult
}

func newFakeSTT() *fakeSTT {
	return &fakeSTT{results: make(chan stt.StreamResult, 16)}
}

func (f *fakeSTT) push(r stt.StreamResult) {
	f.results <- r
}

func (f *fakeSTT) StreamTranscribe(
	ctx context.Context,
	_ <-chan []byte,
	_ ...stt.Option,
) (<-chan stt.StreamResult, error) {
	out := make(chan stt.StreamResult, 16)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case r, ok := <-f.results:
				if !ok {
					return
				}
				select {
				case out <- r:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func (f *fakeSTT) Transcribe(
	context.Context,
	[]byte,
	...stt.Option,
) (*stt.Response, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeSTT) Translate(
	context.Context,
	[]byte,
	...stt.Option,
) (*stt.Response, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeSTT) SupportsStreaming() bool { return true }

func (f *fakeSTT) Model() model.TranscriptionModel { return model.TranscriptionModel{} }

type fakeTTS struct {
	mu       sync.Mutex
	streams  []*fakeTTSStream
	supports bool
}

func newFakeTTS(supportsStreamingText bool) *fakeTTS {
	return &fakeTTS{supports: supportsStreamingText}
}

func (f *fakeTTS) currentStream() *fakeTTSStream {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.streams) == 0 {
		return nil
	}
	return f.streams[len(f.streams)-1]
}

type fakeTTSStream struct {
	textIn <-chan string
	chunks chan tts.Chunk
	closed chan struct{}
}

func (f *fakeTTS) StreamAudioFromText(
	ctx context.Context,
	textIn <-chan string,
	_ ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	if !f.supports {
		return nil, errors.New("streaming text not supported")
	}
	s := &fakeTTSStream{
		textIn: textIn,
		chunks: make(chan tts.Chunk, 16),
		closed: make(chan struct{}),
	}
	f.mu.Lock()
	f.streams = append(f.streams, s)
	f.mu.Unlock()
	go func() {
		defer close(s.chunks)
		defer close(s.closed)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-textIn:
				if !ok {
					return
				}
			}
		}
	}()
	return s.chunks, nil
}

// pushChunk lets a test inject an audio chunk into the most recent TTS stream.
// Returns false if the stream is closed or full.
func (f *fakeTTS) pushChunk(data []byte) bool {
	s := f.currentStream()
	if s == nil {
		return false
	}
	defer func() { _ = recover() }()
	select {
	case <-s.closed:
		return false
	default:
	}
	select {
	case s.chunks <- tts.Chunk{Data: data}:
		return true
	case <-s.closed:
		return false
	default:
		return false
	}
}

func (f *fakeTTS) StreamAudio(
	context.Context,
	string,
	...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	ch := make(chan tts.Chunk)
	close(ch)
	return ch, nil
}

func (f *fakeTTS) GenerateAudio(
	context.Context,
	string,
	...tts.GenerationOption,
) (*tts.Response, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTTS) ListVoices(context.Context) ([]tts.Voice, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTTS) Model() model.AudioModel { return model.AudioModel{} }

type fakeTransport struct {
	mu      sync.Mutex
	written [][]byte
	read    chan []byte
	closed  bool
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{read: make(chan []byte, 16)}
}

func (f *fakeTransport) Read(ctx context.Context) ([]byte, error) {
	select {
	case b, ok := <-f.read:
		if !ok {
			return nil, io.EOF
		}
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *fakeTransport) Write(_ context.Context, frame []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(frame))
	copy(cp, frame)
	f.written = append(f.written, cp)
	return nil
}

func (f *fakeTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.read)
	}
	return nil
}

func (f *fakeTransport) writes() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]byte, len(f.written))
	copy(out, f.written)
	return out
}

// scriptedLLM produces a deterministic stream of llm.Events. Returns a
// function that the fakeLLM can be loaded with.
func scriptedLLM(
	events ...llm.Event,
) func(ctx context.Context) <-chan llm.Event {
	return func(ctx context.Context) <-chan llm.Event {
		ch := make(chan llm.Event, len(events)+1)
		go func() {
			defer close(ch)
			for _, e := range events {
				select {
				case <-ctx.Done():
					ch <- llm.Event{Type: types.EventError, Error: ctx.Err()}
					return
				case ch <- e:
				}
			}
		}()
		return ch
	}
}
