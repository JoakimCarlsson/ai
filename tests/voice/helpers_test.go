// Public-API black-box helpers for tests/voice. The fakes below mirror the
// private helpers in voice/helpers_test.go but implement only the exported
// interfaces so this package can compile against the public surface.
package voice

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/tts"
	"github.com/joakimcarlsson/ai/types"
	"github.com/joakimcarlsson/ai/voice"
)

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for: %s", msg)
}

// scriptedLLM produces a deterministic stream of llm.Events.
func scriptedLLM(events ...llm.Event) func(ctx context.Context) <-chan llm.Event {
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

type fakeLLM struct {
	mu        sync.Mutex
	id        string
	calls     int
	scripts   []func(ctx context.Context) <-chan llm.Event
	lastMsgs  []message.Message
	lastTools []tool.BaseTool
}

func newFakeLLM(id string) *fakeLLM { return &fakeLLM{id: id} }

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

func (f *fakeLLM) lastToolList() []tool.BaseTool {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]tool.BaseTool, len(f.lastTools))
	copy(out, f.lastTools)
	return out
}

func (f *fakeLLM) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeLLM) StreamResponse(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	f.mu.Lock()
	f.lastMsgs = make([]message.Message, len(msgs))
	copy(f.lastMsgs, msgs)
	f.lastTools = make([]tool.BaseTool, len(tools))
	copy(f.lastTools, tools)
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

func (f *fakeLLM) Model() model.Model               { return model.Model{} }
func (f *fakeLLM) SupportsStructuredOutput() bool   { return false }

type fakeSTT struct {
	results chan stt.StreamResult
}

func newFakeSTT() *fakeSTT {
	return &fakeSTT{results: make(chan stt.StreamResult, 16)}
}

func (f *fakeSTT) push(r stt.StreamResult)    { f.results <- r }
func (f *fakeSTT) pushFinal(text string)       { f.push(stt.StreamResult{Text: text, IsFinal: true}) }
func (f *fakeSTT) pushPartial(text string)     { f.push(stt.StreamResult{Text: text, IsFinal: false}) }

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

func (f *fakeSTT) SupportsStreaming() bool         { return true }
func (f *fakeSTT) Model() model.TranscriptionModel { return model.TranscriptionModel{} }

type fakeTTS struct {
	mu       sync.Mutex
	streams  []*fakeTTSStream
	supports bool
}

func newFakeTTS(supportsStreamingText bool) *fakeTTS {
	return &fakeTTS{supports: supportsStreamingText}
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

// fakeTool records its last received input and returns a configurable
// output / error. Implements tool.BaseTool via the public package.
type fakeTool struct {
	mu        sync.Mutex
	name      string
	desc      string
	lastInput string
	output    string
	err       error
}

func newFakeTool(name string) *fakeTool {
	return &fakeTool{name: name, desc: "fake tool", output: "ok"}
}

func (f *fakeTool) setOutput(out string)       { f.mu.Lock(); defer f.mu.Unlock(); f.output = out }
func (f *fakeTool) setError(err error)         { f.mu.Lock(); defer f.mu.Unlock(); f.err = err }
func (f *fakeTool) lastReceivedInput() string  { f.mu.Lock(); defer f.mu.Unlock(); return f.lastInput }

func (f *fakeTool) Info() tool.Info {
	return tool.Info{
		Name:        f.name,
		Description: f.desc,
		Parameters:  map[string]any{"type": "object"},
	}
}

func (f *fakeTool) Run(_ context.Context, c tool.Call) (tool.Response, error) {
	f.mu.Lock()
	f.lastInput = c.Input
	out := f.output
	err := f.err
	f.mu.Unlock()
	if err != nil {
		return tool.NewTextErrorResponse(err.Error()), err
	}
	return tool.NewTextResponse(out), nil
}

// testAgent wraps a Conversation plus the fakes so tests can drive it
// through the public API.
type testAgent struct {
	conv      *voice.Conversation
	stt       *fakeSTT
	tts       *fakeTTS
	transport *fakeTransport
	cancel    context.CancelFunc

	mu     sync.Mutex
	events []voice.Event
}

func newTestAgent(
	t *testing.T,
	llmFake *fakeLLM,
	opts ...voice.Option,
) *testAgent {
	t.Helper()
	sttFake := newFakeSTT()
	ttsFake := newFakeTTS(true)
	v := voice.New(llmFake, sttFake, ttsFake, opts...)

	ctx, cancel := context.WithCancel(context.Background())
	transport := newFakeTransport()
	conv, err := v.StartConversation(ctx, transport)
	if err != nil {
		cancel()
		t.Fatalf("StartConversation: %v", err)
	}
	a := &testAgent{
		conv:      conv,
		stt:       sttFake,
		tts:       ttsFake,
		transport: transport,
		cancel:    cancel,
	}
	go func() {
		for evt := range conv.Events() {
			a.mu.Lock()
			a.events = append(a.events, evt)
			a.mu.Unlock()
		}
	}()
	return a
}

func (a *testAgent) hasEvent(t voice.EventType) bool {
	return a.countEvents(t) > 0
}

func (a *testAgent) countEvents(t voice.EventType) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for _, e := range a.events {
		if e.Type == t {
			n++
		}
	}
	return n
}

func (a *testAgent) eventsOfType(t voice.EventType) []voice.Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []voice.Event
	for _, e := range a.events {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out
}
