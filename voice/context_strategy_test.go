package voice

import (
	"context"
	"errors"
	"sync"
	"testing"

	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/types"
)

// fakeStrategy records its inputs and returns a configurable result. Used
// to verify the runner invokes Fit with the right arguments and acts on the
// returned trimmed message list.
type fakeStrategy struct {
	mu        sync.Mutex
	calls     int
	lastInput tokens.StrategyInput
	result    *tokens.StrategyResult
	err       error
}

func (f *fakeStrategy) Fit(
	_ context.Context,
	in tokens.StrategyInput,
) (*tokens.StrategyResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastInput = in
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &tokens.StrategyResult{Messages: in.Messages}, nil
}

func (f *fakeStrategy) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// Strategy is invoked before each LLM call with the agent's system prompt,
// tools, and the current message history.
func TestContextStrategy_FitCalledWithExpectedInput(t *testing.T) {
	strategy := &fakeStrategy{}
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "ok. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("you are helpful"),
		WithContextStrategy(strategy, 8000),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if strategy.callCount() != 1 {
		t.Fatalf("expected Fit called once, got %d", strategy.callCount())
	}
	strategy.mu.Lock()
	in := strategy.lastInput
	strategy.mu.Unlock()

	if in.SystemPrompt != "you are helpful" {
		t.Fatalf(
			"expected SystemPrompt='you are helpful', got %q",
			in.SystemPrompt,
		)
	}
	if in.MaxTokens != 8000 {
		t.Fatalf("expected MaxTokens=8000, got %d", in.MaxTokens)
	}
	if in.Counter == nil {
		t.Fatalf("expected non-nil Counter")
	}
	if len(in.Messages) == 0 {
		t.Fatalf("expected non-empty Messages")
	}
}

// When the strategy returns a trimmed message list, that list is what
// reaches the LLM, not the full history.
func TestContextStrategy_TrimmedMessagesReachLLM(t *testing.T) {
	strategy := &fakeStrategy{
		result: &tokens.StrategyResult{
			Messages: []message.Message{
				message.NewUserMessage("trimmed-only"),
			},
		},
	}
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "ok. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("sys"),
		WithContextStrategy(strategy, 8000),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "long question", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	got := llmFake.lastMessages()
	if len(got) != 1 {
		t.Fatalf(
			"expected LLM to receive 1 trimmed message, got %d: %+v",
			len(got),
			got,
		)
	}
	for _, p := range got[0].Parts {
		if tc, ok := p.(message.TextContent); ok && tc.Text != "trimmed-only" {
			t.Fatalf("expected trimmed-only text, got %q", tc.Text)
		}
	}
}

// SessionUpdate.AddMessages from the strategy is folded into the live
// history and persisted alongside the rest of the turn's new messages by
// the runner's per-turn persist step.
func TestContextStrategy_SessionUpdatePersisted(t *testing.T) {
	summary := message.NewMessage(
		message.Summary,
		[]message.ContentPart{
			message.TextContent{Text: "summary of older turns"},
		},
	)

	strategy := &fakeStrategy{
		result: &tokens.StrategyResult{
			Messages: []message.Message{
				message.NewUserMessage("just the new ask"),
			},
			SessionUpdate: &tokens.SessionUpdate{
				AddMessages: []message.Message{summary},
			},
		},
	}

	store := session.MemoryStore()
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "ok. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("sys"),
		WithSession("sess-ctx", store),
		WithContextStrategy(strategy, 8000),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "ask", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	sess, _ := store.Load(context.Background(), "sess-ctx")
	msgs, _ := sess.GetMessages(context.Background(), nil)

	var sawSummary bool
	for _, m := range msgs {
		if m.Role == message.Summary {
			for _, p := range m.Parts {
				if tc, ok := p.(message.TextContent); ok &&
					tc.Text == "summary of older turns" {
					sawSummary = true
				}
			}
		}
	}
	if !sawSummary {
		t.Fatalf(
			"expected summary message persisted to session; got %d msgs: %+v",
			len(msgs),
			msgs,
		)
	}
}

// applySessionUpdate honors SessionUpdate.PopCount: messages already persisted
// to the session store are popped (so the next flush does not duplicate the
// retained tail) and sessionPersisted is rewound to match. Mirrors the agent
// buildMessages fix (#199).
func TestApplySessionUpdate_PopCountRewindsStore(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()
	sess, err := store.Create(ctx, "sess-pop")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	u1 := message.NewUserMessage("u1")
	a1 := message.NewMessage(
		message.Assistant,
		[]message.ContentPart{message.TextContent{Text: "a1"}},
	)
	u2 := message.NewUserMessage("u2")
	if err := sess.AddMessages(ctx, []message.Message{u1, a1, u2}); err != nil {
		t.Fatalf("seed messages: %v", err)
	}

	v := &Agent{session: sess}
	history := []message.Message{u1, a1, u2}
	sessionPersisted := len(history)

	summary := message.NewMessage(
		message.Summary,
		[]message.ContentPart{message.TextContent{Text: "summary"}},
	)
	update := &tokens.SessionUpdate{
		PopCount:    2,
		AddMessages: []message.Message{summary, u2},
	}

	if err := applySessionUpdate(
		ctx, v, &history, &sessionPersisted, update,
	); err != nil {
		t.Fatalf("applySessionUpdate: %v", err)
	}

	if sessionPersisted != 1 {
		t.Fatalf("expected sessionPersisted rewound to 1, got %d", sessionPersisted)
	}
	if stored, _ := sess.GetMessages(ctx, nil); len(stored) != 1 {
		t.Fatalf("expected 1 message left in store after pop, got %d", len(stored))
	}
	if len(history) != 3 || history[1].Role != message.Summary {
		t.Fatalf("expected history [u1, summary, u2], got %+v", history)
	}

	if err := sess.AddMessages(ctx, history[sessionPersisted:]); err != nil {
		t.Fatalf("flush: %v", err)
	}
	final, _ := sess.GetMessages(ctx, nil)
	if len(final) != 3 {
		t.Fatalf(
			"expected 3 messages after flush (no duplication), got %d: %+v",
			len(final),
			final,
		)
	}
	summaries := 0
	for _, m := range final {
		if m.Role == message.Summary {
			summaries++
		}
	}
	if summaries != 1 {
		t.Fatalf("expected exactly 1 summary after flush, got %d", summaries)
	}
}

// When no strategy is configured, history is passed through to the LLM
// untouched (regression check for the no-op path).
func TestContextStrategy_NoStrategyPassesThrough(t *testing.T) {
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "ok. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("sys"),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	got := llmFake.lastMessages()
	if len(got) < 2 {
		t.Fatalf(
			"expected at least system+user msgs at LLM, got %d: %+v",
			len(got),
			got,
		)
	}
}

// Strategy errors propagate as conversation errors.
func TestContextStrategy_FitErrorEndsConversation(t *testing.T) {
	wantErr := errors.New("strategy boom")
	strategy := &fakeStrategy{err: wantErr}
	llmFake := &fakeLLM{}

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("sys"),
		WithContextStrategy(strategy, 8000),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: true})

	err := a.conv.Wait()
	if err == nil {
		t.Fatalf("expected error from strategy; got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
}
