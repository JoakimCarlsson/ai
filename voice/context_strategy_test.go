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
	strat := &fakeStrategy{}
	llmFake := &fakeLLM{}
	llmFake.push(scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: "ok. "},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	))

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("you are helpful"),
		WithContextStrategy(strat, 8000),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "hi", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if strat.callCount() != 1 {
		t.Fatalf("expected Fit called once, got %d", strat.callCount())
	}
	strat.mu.Lock()
	in := strat.lastInput
	strat.mu.Unlock()

	if in.SystemPrompt != "you are helpful" {
		t.Fatalf("expected SystemPrompt='you are helpful', got %q", in.SystemPrompt)
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
	strat := &fakeStrategy{
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
		WithContextStrategy(strat, 8000),
	)
	defer a.cancel()

	a.stt.push(stt.StreamResult{Text: "long question", IsFinal: true})
	waitFor(t, func() bool { return a.hasEvent(EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	got := llmFake.lastMessages()
	if len(got) != 1 {
		t.Fatalf("expected LLM to receive 1 trimmed message, got %d: %+v", len(got), got)
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
	summary := message.NewMessage(message.Summary,
		[]message.ContentPart{message.TextContent{Text: "summary of older turns"}})

	strat := &fakeStrategy{
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
		WithContextStrategy(strat, 8000),
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
		t.Fatalf("expected summary message persisted to session; got %d msgs: %+v",
			len(msgs), msgs)
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
		t.Fatalf("expected at least system+user msgs at LLM, got %d: %+v", len(got), got)
	}
}

// Strategy errors propagate as conversation errors.
func TestContextStrategy_FitErrorEndsConversation(t *testing.T) {
	wantErr := errors.New("strategy boom")
	strat := &fakeStrategy{err: wantErr}
	llmFake := &fakeLLM{}

	a := newTestAgent(t, llmFake,
		WithSystemPrompt("sys"),
		WithContextStrategy(strat, 8000),
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
