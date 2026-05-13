package voice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/tts"
	"github.com/joakimcarlsson/ai/voice"
)

// 1. With WithInitialMessage set, the agent speaks the greeting on a fresh
// conversation. TTS is invoked with the configured text, the standard
// event sequence fires, and no LLM call happens.
func TestInitialMessage_SpeaksOnFreshConversation(t *testing.T) {
	llmFake := newFakeLLM("init-speaks")
	a := newTestAgent(t, llmFake, voice.WithInitialMessage("hello there"))
	defer a.cancel()

	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done after initial message")

	deltas := a.eventsOfType(voice.EventAssistantDelta)
	if len(deltas) == 0 || deltas[0].Text != "hello there" {
		t.Fatalf("expected delta with greeting text; got %+v", deltas)
	}
	if !a.hasEvent(voice.EventTTSStarted) {
		t.Fatalf("expected EventTTSStarted before greeting")
	}
	if !a.hasEvent(voice.EventTTSEnded) {
		t.Fatalf("expected EventTTSEnded after greeting")
	}
	if llmFake.callCount() != 0 {
		t.Fatalf("LLM should not be called for initial message; got %d calls",
			llmFake.callCount())
	}

	a.cancel()
	_ = a.conv.Wait()
}

// 2. When resuming a session that already has non-system messages, the
// greeting is skipped — the user is mid-thread and should not hear a
// fresh hello.
func TestInitialMessage_SkippedOnSessionResume(t *testing.T) {
	store := session.MemoryStore()
	ctx := context.Background()
	sess, _ := store.Create(ctx, "sess-resume")
	_ = sess.AddMessages(ctx, []message.Message{
		message.NewSystemMessage("you are helpful"),
		message.NewUserMessage("earlier ask"),
		message.NewMessage(message.Assistant,
			[]message.ContentPart{message.TextContent{Text: "earlier reply"}}),
	})

	llmFake := newFakeLLM("init-skip-resume")
	a := newTestAgent(t, llmFake,
		voice.WithInitialMessage("hello again"),
		voice.WithSystemPrompt("you are helpful"),
		voice.WithSession("sess-resume", store),
	)
	defer a.cancel()

	waitFor(t, func() bool { return a.hasEvent(voice.EventReady) },
		"ready")

	// 100ms grace: if a greeting were going to fire it would have by now.
	time.Sleep(100 * time.Millisecond)
	if a.hasEvent(voice.EventTTSStarted) {
		t.Fatalf("expected greeting to be skipped on session resume; "+
			"got events %+v", a.eventsOfType(voice.EventTTSStarted))
	}

	a.cancel()
	_ = a.conv.Wait()
}

// 3. The greeting is appended to the session as an assistant message.
func TestInitialMessage_PersistsToSession(t *testing.T) {
	store := session.MemoryStore()
	llmFake := newFakeLLM("init-persist")
	a := newTestAgent(t, llmFake,
		voice.WithInitialMessage("welcome"),
		voice.WithSession("sess-init-persist", store),
	)
	defer a.cancel()

	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	sess, _ := store.Load(context.Background(), "sess-init-persist")
	msgs, _ := sess.GetMessages(context.Background(), nil)
	if !anyAssistantText(msgs, "welcome") {
		t.Fatalf("expected greeting persisted to session; got %+v", msgs)
	}
}

// 4. Empty WithInitialMessage (or unset) is a no-op: no TTS, no events.
func TestInitialMessage_EmptyIsNoOp(t *testing.T) {
	llmFake := newFakeLLM("init-empty")
	a := newTestAgent(t, llmFake, voice.WithInitialMessage(""))
	defer a.cancel()

	waitFor(t, func() bool { return a.hasEvent(voice.EventReady) },
		"ready")

	time.Sleep(100 * time.Millisecond)
	if a.hasEvent(voice.EventTTSStarted) ||
		a.hasEvent(voice.EventAssistantDone) {
		t.Fatalf("expected no greeting events with empty initial message")
	}

	a.cancel()
	_ = a.conv.Wait()
}

// 5. When the TTS client does not implement tts.StreamingTextProvider, the
// greeting falls back to tts.Generation.StreamAudio (speakOneShot).
func TestInitialMessage_NonStreamingFallback(t *testing.T) {
	nstts := &nonStreamingTTS{}
	llmFake := newFakeLLM("init-nonstream")
	sttFake := newFakeSTT()
	v := voice.New(llmFake, sttFake, nstts,
		voice.WithInitialMessage("hi via one-shot"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport := newFakeTransport()
	conv, err := v.StartConversation(ctx, transport)
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}

	var (
		mu     sync.Mutex
		events []voice.Event
	)
	go func() {
		for evt := range conv.Events() {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		}
	}()

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, e := range events {
			if e.Type == voice.EventAssistantDone {
				return true
			}
		}
		return false
	}, "assistant done via non-streaming")

	mu.Lock()
	defer mu.Unlock()
	if nstts.calls() != 1 {
		t.Fatalf("expected StreamAudio called once for fallback; got %d",
			nstts.calls())
	}
	if got := nstts.lastText(); got != "hi via one-shot" {
		t.Fatalf("expected StreamAudio text 'hi via one-shot'; got %q", got)
	}
}

// 6. Under BargeInInterrupt, a user partial during the greeting cancels TTS
// and the assistant history entry is recorded with an "[interrupted]"
// suffix. Uses a custom TTS that holds audio open until ctx cancels so the
// barge-in window exists.
func TestInitialMessage_BargeInTruncatesAndPersists(t *testing.T) {
	store := session.MemoryStore()
	holdingTTS := newHoldingTTS()

	llmFake := newFakeLLM("init-barge")
	sttFake := newFakeSTT()
	v := voice.New(llmFake, sttFake, holdingTTS,
		voice.WithInitialMessage("welcome to the agent"),
		voice.WithBargeIn(voice.BargeInInterrupt),
		voice.WithSession("sess-barge", store),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport := newFakeTransport()
	conv, err := v.StartConversation(ctx, transport)
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}

	var (
		mu     sync.Mutex
		events []voice.Event
	)
	go func() {
		for evt := range conv.Events() {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		}
	}()

	hasEvt := func(want voice.EventType) bool {
		mu.Lock()
		defer mu.Unlock()
		for _, e := range events {
			if e.Type == want {
				return true
			}
		}
		return false
	}

	waitFor(t, func() bool { return hasEvt(voice.EventTTSStarted) },
		"TTS started for greeting")

	sttFake.push(stt.StreamResult{Text: "stop", IsFinal: false})

	waitFor(t, func() bool { return hasEvt(voice.EventAgentInterrupted) },
		"EventAgentInterrupted fired during greeting")

	waitFor(t, func() bool {
		sess, _ := store.Load(context.Background(), "sess-barge")
		msgs, _ := sess.GetMessages(context.Background(), nil)
		return anyAssistantText(msgs, "[interrupted]")
	}, "greeting persisted with [interrupted] suffix")

	cancel()
	_ = conv.Wait()
}

// anyAssistantText reports whether any persisted assistant message
// contains the given substring in a TextContent part.
func anyAssistantText(msgs []message.Message, want string) bool {
	for _, m := range msgs {
		if m.Role != message.Assistant {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok &&
				strings.Contains(tc.Text, want) {
				return true
			}
		}
	}
	return false
}

// nonStreamingTTS implements tts.Generation but NOT
// tts.StreamingTextProvider, so the initial-message path falls back to
// speakOneShot.
type nonStreamingTTS struct {
	mu        sync.Mutex
	callCount int
	last      string
}

func (n *nonStreamingTTS) calls() int { n.mu.Lock(); defer n.mu.Unlock(); return n.callCount }
func (n *nonStreamingTTS) lastText() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.last
}

func (n *nonStreamingTTS) StreamAudio(
	_ context.Context,
	text string,
	_ ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	n.mu.Lock()
	n.callCount++
	n.last = text
	n.mu.Unlock()
	ch := make(chan tts.Chunk, 1)
	ch <- tts.Chunk{Data: []byte{0x00}}
	close(ch)
	return ch, nil
}

func (n *nonStreamingTTS) GenerateAudio(
	context.Context,
	string,
	...tts.GenerationOption,
) (*tts.Response, error) {
	return nil, errors.New("not implemented")
}

func (n *nonStreamingTTS) ListVoices(context.Context) ([]tts.Voice, error) {
	return nil, errors.New("not implemented")
}

func (n *nonStreamingTTS) Model() model.AudioModel { return model.AudioModel{} }

// holdingTTS implements tts.StreamingTextProvider but keeps the audio
// output channel open until ctx cancels, so a barge-in test has a
// window to inject the interruption while the agent is in the audio
// pump loop.
type holdingTTS struct {
	mu       sync.Mutex
	received []string
}

func newHoldingTTS() *holdingTTS { return &holdingTTS{} }

func (h *holdingTTS) StreamAudioFromText(
	ctx context.Context,
	textIn <-chan string,
	_ ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	out := make(chan tts.Chunk, 16)
	go func() {
		defer close(out)
		// Drain textIn until closed, then block on ctx so the audio
		// pump in speakInitialMessage stays open and barge-in has time
		// to fire.
		for {
			select {
			case <-ctx.Done():
				return
			case text, ok := <-textIn:
				if !ok {
					<-ctx.Done()
					return
				}
				h.mu.Lock()
				h.received = append(h.received, text)
				h.mu.Unlock()
			}
		}
	}()
	return out, nil
}

func (h *holdingTTS) StreamAudio(
	context.Context,
	string,
	...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	ch := make(chan tts.Chunk)
	close(ch)
	return ch, nil
}

func (h *holdingTTS) GenerateAudio(
	context.Context,
	string,
	...tts.GenerationOption,
) (*tts.Response, error) {
	return nil, errors.New("not implemented")
}

func (h *holdingTTS) ListVoices(context.Context) ([]tts.Voice, error) {
	return nil, errors.New("not implemented")
}

func (h *holdingTTS) Model() model.AudioModel { return model.AudioModel{} }
