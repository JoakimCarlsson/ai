package voice

import (
	"context"
	"strings"
	"testing"
	"time"

	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/types"
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

// streamThenHold emits the given content deltas and then blocks on the hold
// channel before emitting a completion event. Lets a test pin the assistant
// in mid-turn so barge-in can fire.
func streamThenHold(deltas []string, hold <-chan struct{}) func(ctx context.Context) <-chan llm.Event {
	return func(ctx context.Context) <-chan llm.Event {
		ch := make(chan llm.Event, len(deltas)+2)
		go func() {
			defer close(ch)
			for _, d := range deltas {
				select {
				case <-ctx.Done():
					return
				case ch <- llm.Event{Type: types.EventContentDelta, Content: d}:
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-hold:
				select {
				case ch <- llm.Event{Type: types.EventComplete, Response: &llm.Response{}}:
				case <-ctx.Done():
				}
			}
		}()
		return ch
	}
}

type testAgent struct {
	conv      *Conversation
	stt       *fakeSTT
	tts       *fakeTTS
	transport *fakeTransport
	cancel    context.CancelFunc
}

func newTestAgent(t *testing.T, llmFake *fakeLLM, opts ...Option) *testAgent {
	t.Helper()
	sttFake := newFakeSTT()
	ttsFake := newFakeTTS(true)
	v := New(llmFake, sttFake, ttsFake, opts...)

	ctx, cancel := context.WithCancel(context.Background())
	transport := newFakeTransport()
	conv, err := v.StartConversation(ctx, transport)
	if err != nil {
		cancel()
		t.Fatalf("StartConversation: %v", err)
	}
	go func() {
		for range conv.Events() {
		}
	}()
	return &testAgent{
		conv:      conv,
		stt:       sttFake,
		tts:       ttsFake,
		transport: transport,
		cancel:    cancel,
	}
}

func TestBargeIn_CancelsAndUpdatesSpokenSoFar(t *testing.T) {
	hold := make(chan struct{})
	llmFake := &fakeLLM{}
	llmFake.push(streamThenHold(
		[]string{"Once upon a time, ", "there was a duck."},
		hold,
	))

	a := newTestAgent(t, llmFake, WithBargeIn(BargeInInterrupt))
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "tell me a story", IsFinal: true})

	waitFor(t, func() bool { return a.tts.currentStream() != nil }, "TTS opened")
	waitFor(t, func() bool {
		s := a.conv.turnState()
		return s != nil && s.agentSpeaking.Load()
	}, "agentSpeaking set")
	waitFor(t, func() bool {
		s := a.conv.turnState()
		return s != nil && strings.Contains(s.loadSpoken(), "Once upon a time")
	}, "spokenSoFar accumulated")

	a.stt.push(stt.StreamResult{Text: "stop", IsFinal: false})

	waitFor(t, func() bool {
		s := a.conv.turnState()
		return s != nil && s.dropAudio.Load()
	}, "dropAudio set after barge-in")

	close(hold)
	a.cancel()
	_ = a.conv.Wait()
}

func TestBargeIn_DoesNotFireWhenAgentNotSpeaking(t *testing.T) {
	llmFake := &fakeLLM{}
	a := newTestAgent(t, llmFake, WithBargeIn(BargeInInterrupt))
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: false})

	time.Sleep(50 * time.Millisecond)
	s := a.conv.turnState()
	if s != nil && s.dropAudio.Load() {
		t.Fatalf("dropAudio set unexpectedly while agent not speaking")
	}
	a.cancel()
	_ = a.conv.Wait()
}

func TestBargeIn_RepeatedInterruptsDoNotLeak(t *testing.T) {
	const N = 5
	holds := make([]chan struct{}, N+1)
	for i := range holds {
		holds[i] = make(chan struct{})
	}
	llmFake := &fakeLLM{}
	for i := range holds {
		llmFake.push(streamThenHold([]string{"reply ", string(rune('A' + i)), ". "}, holds[i]))
	}

	a := newTestAgent(t, llmFake, WithBargeIn(BargeInInterrupt))
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "ask", IsFinal: true})

	for i := range N {
		waitFor(t, func() bool {
			s := a.conv.turnState()
			return s != nil && s.agentSpeaking.Load()
		}, "agentSpeaking turn "+string(rune('0'+i)))

		a.stt.push(stt.StreamResult{Text: "stop", IsFinal: false})
		waitFor(t, func() bool {
			s := a.conv.turnState()
			return s != nil && s.dropAudio.Load()
		}, "dropAudio set turn "+string(rune('0'+i)))

		junk := []byte{byte(0xF0 + i), 0xAA, 0xBB, 0xCC}
		for range 3 {
			a.tts.pushChunk(junk)
		}

		close(holds[i])
		a.stt.push(stt.StreamResult{Text: "next", IsFinal: true})

		waitFor(t, func() bool {
			s := a.conv.turnState()
			return s != nil && !s.dropAudio.Load()
		}, "dropAudio reset turn "+string(rune('0'+i)))
	}

	close(holds[N])
	a.cancel()
	_ = a.conv.Wait()

	for i, w := range a.transport.writes() {
		if len(w) >= 4 && w[0] >= 0xF0 && w[1] == 0xAA && w[2] == 0xBB && w[3] == 0xCC {
			t.Fatalf("interrupted-turn junk leaked to transport at write %d: %v", i, w)
		}
	}
}

func TestBargeIn_StaleAudioDoesNotLeakIntoNextTurn(t *testing.T) {
	hold1 := make(chan struct{})
	hold2 := make(chan struct{})
	llmFake := &fakeLLM{}
	llmFake.push(streamThenHold([]string{"First reply. "}, hold1))
	llmFake.push(streamThenHold([]string{"Second reply. "}, hold2))

	a := newTestAgent(t, llmFake, WithBargeIn(BargeInInterrupt))
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "first ask", IsFinal: true})
	waitFor(t, func() bool { return a.tts.currentStream() != nil }, "turn 1 TTS opened")
	waitFor(t, func() bool {
		s := a.conv.turnState()
		return s != nil && s.agentSpeaking.Load()
	}, "turn 1 agentSpeaking")

	a.stt.push(stt.StreamResult{Text: "actually", IsFinal: false})
	waitFor(t, func() bool {
		s := a.conv.turnState()
		return s != nil && s.dropAudio.Load()
	}, "barge-in fires")

	sentinel := []byte{0x01, 0x02, 0x03, 0x04}
	for range 5 {
		a.tts.pushChunk(sentinel)
	}

	close(hold1)
	a.stt.push(stt.StreamResult{Text: "actually a poem", IsFinal: true})

	waitFor(t, func() bool {
		s := a.conv.turnState()
		return s != nil && !s.dropAudio.Load()
	}, "dropAudio reset for turn 2")

	close(hold2)
	a.cancel()
	_ = a.conv.Wait()

	for i, w := range a.transport.writes() {
		if len(w) >= 4 && w[0] == 0x01 && w[1] == 0x02 && w[2] == 0x03 && w[3] == 0x04 {
			t.Fatalf("sentinel turn-1 stale audio leaked to transport at write %d: %v", i, w)
		}
	}
}
