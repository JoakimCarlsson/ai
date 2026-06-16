package voice

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/rag"
	"github.com/joakimcarlsson/ai/voice"
)

// fakeKB is a minimal rag.KnowledgeBase for voice tests. It records
// Retrieve calls so we can assert how often recall fires across LLM
// iterations within the same turn.
type fakeKB struct {
	mu      sync.Mutex
	id      string
	hits    []rag.Hit
	calls   int
	lastQry string
}

func (k *fakeKB) ID() string { return k.id }

func (k *fakeKB) Ingest(_ context.Context, _ []rag.Document) error {
	return nil
}

func (k *fakeKB) Retrieve(
	_ context.Context,
	query string,
	_ int,
) ([]rag.Hit, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.calls++
	k.lastQry = query
	out := make([]rag.Hit, len(k.hits))
	copy(out, k.hits)
	return out, nil
}

func (k *fakeKB) callCount() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.calls
}

// TestKB_RecallInjectsSystemMessage verifies that voice.WithKnowledgeBase
// prepends a "Relevant context from the knowledge base:" system message
// to the LLM input on each user turn.
func TestKB_RecallInjectsSystemMessage(t *testing.T) {
	kb := &fakeKB{
		id: "docs",
		hits: []rag.Hit{
			{
				Chunk: rag.Chunk{
					DocumentID: "policy-returns",
					Content:    "Returns within 30 days for a full refund.",
				},
				Score: 0.9,
			},
		},
	}

	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithSystemPrompt("you are helpful"),
		voice.WithKnowledgeBase(kb),
	)
	defer a.cancel()

	a.stt.pushFinal("what is the return policy?")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if kb.callCount() < 1 {
		t.Fatalf("expected KB.Retrieve called >= 1, got %d", kb.callCount())
	}

	msgs := llmFake.lastMessages()
	var sawRecall bool
	for _, m := range msgs {
		if m.Role != message.System {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok &&
				strings.Contains(
					tc.Text,
					"Relevant context from the knowledge base",
				) &&
				strings.Contains(tc.Text, "policy-returns") {
				sawRecall = true
			}
		}
	}
	if !sawRecall {
		t.Fatalf("expected system message with KB context; got %+v", msgs)
	}
}

// TestKB_NoKBNoEffect is a regression check that omitting
// WithKnowledgeBase produces no KB system message.
func TestKB_NoKBNoEffect(t *testing.T) {
	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithSystemPrompt("sys"),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	for _, m := range llmFake.lastMessages() {
		if m.Role != message.System {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok &&
				strings.Contains(
					tc.Text,
					"Relevant context from the knowledge base",
				) {
				t.Fatalf(
					"unexpected KB header without WithKnowledgeBase: %q",
					tc.Text,
				)
			}
		}
	}
}
