package voice

import (
	"context"
	"strings"
	"testing"

	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/types"
)

// hasAssistantTextContaining returns true if any persisted assistant message
// holds a TextContent part with the given substring.
func hasAssistantTextContaining(msgs []message.Message, want string) bool {
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

// When the session is empty and a system prompt is set, the runner persists
// the system prompt as the first message.
func TestSession_PersistsSystemPromptOnFreshSession(t *testing.T) {
	store := session.MemoryStore()
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "Hi. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("you are helpful"),
		WithSession("sess-fresh", store),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	sess, _ := store.Load(context.Background(), "sess-fresh")
	msgs, _ := sess.GetMessages(context.Background(), nil)
	if len(msgs) == 0 || msgs[0].Role != message.System {
		t.Fatalf("expected first persisted msg to be system; got %d msgs: %+v",
			len(msgs), msgs)
	}
}

// When the session already holds messages, the runner uses them as the
// starting history and does NOT re-persist the system prompt. New turn
// messages are appended after the existing ones.
func TestSession_LoadsExistingMessages(t *testing.T) {
	store := session.MemoryStore()
	ctx := context.Background()

	sess, _ := store.Create(ctx, "sess-warm")
	_ = sess.AddMessages(ctx, []message.Message{
		message.NewSystemMessage("you are helpful"),
		message.NewUserMessage("earlier ask"),
		message.NewMessage(message.Assistant,
			[]message.ContentPart{message.TextContent{Text: "earlier reply"}}),
	})

	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "ok. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("you are helpful"),
		WithSession("sess-warm", store),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "next ask", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	msgs, _ := sess.GetMessages(ctx, nil)
	if len(msgs) != 5 {
		t.Fatalf("expected 5 msgs (3 prior + user + assistant); got %d: %+v",
			len(msgs), msgs)
	}
	if msgs[3].Role != message.User {
		t.Fatalf("expected msgs[3]=user, got %v", msgs[3].Role)
	}
	if msgs[4].Role != message.Assistant {
		t.Fatalf("expected msgs[4]=assistant, got %v", msgs[4].Role)
	}
}

// On a normal turn end with no tool calls, the assistant's spoken text is
// persisted to the session. This covers the pipeline.go bug fix.
func TestSession_PersistsAssistantTextWithoutTools(t *testing.T) {
	store := session.MemoryStore()
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "Hi back. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSession("sess-text", store),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	sess, _ := store.Load(context.Background(), "sess-text")
	msgs, _ := sess.GetMessages(context.Background(), nil)
	if !hasAssistantTextContaining(msgs, "Hi back") {
		t.Fatalf("expected assistant text 'Hi back' persisted; got %+v", msgs)
	}
}

// On barge-in, the truncated assistant reply (with [interrupted] suffix) is
// persisted to the session.
func TestSession_PersistsInterruptedReply(t *testing.T) {
	store := session.MemoryStore()
	hold := make(chan struct{})
	llmFake := &fakeLLM{}
	llmFake.push(streamThenHold(
		[]string{"telling a long story. "},
		hold,
	))

	a := newTestAgent(t, llmFake,
		WithBargeIn(BargeInInterrupt),
		WithSession("sess-int", store),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "story", IsFinal: true})
	waitFor(t, func() bool {
		s := a.conv.turnState()
		return s != nil && s.agentSpeaking.Load()
	}, "speaking")

	a.stt.push(stt.StreamResult{Text: "stop", IsFinal: false})
	waitFor(t, func() bool { return a.hasEvent(EventAgentInterrupted) },
		"interrupted event fired")

	waitFor(t, func() bool {
		sess, _ := store.Load(context.Background(), "sess-int")
		msgs, _ := sess.GetMessages(context.Background(), nil)
		return hasAssistantTextContaining(msgs, "[interrupted]")
	}, "interrupted reply persisted")

	close(hold)
	a.cancel()
	_ = a.conv.Wait()
}

// Without WithSession, no persistence happens — history stays in-memory and
// the conversation still works. This pins the no-session contract.
func TestSession_NoSessionStillWorks(t *testing.T) {
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "Hi. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("you are helpful"),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()
}
