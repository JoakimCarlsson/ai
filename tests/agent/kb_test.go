package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/rag"
)

// fakeKB satisfies rag.KnowledgeBase without exercising any real
// embedding or store. Search results are seeded directly so we can
// assert on the system message buildMessages produces.
type fakeKB struct {
	id      string
	hits    []rag.Hit
	calls   int
	lastQry string
	err     error
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
	k.calls++
	k.lastQry = query
	if k.err != nil {
		return nil, k.err
	}
	return k.hits, nil
}

// TestKnowledgeBase_RetrieveInjectsSystemContext verifies that
// agent.WithKnowledgeBase causes Retrieve to fire and the result to
// land in the system prompt sent to the LLM.
func TestKnowledgeBase_RetrieveInjectsSystemContext(t *testing.T) {
	t.Parallel()

	kb := &fakeKB{
		id: "docs",
		hits: []rag.Hit{
			{
				Chunk: rag.Chunk{
					DocumentID: "policy-returns",
					Content:    "Returns are accepted within 30 days.",
				},
				Score: 0.9,
			},
		},
	}

	mock := newMockLLM(mockResponse{Content: "ok"})
	a := agent.New(mock,
		agent.WithSystemPrompt("you are helpful"),
		agent.WithKnowledgeBase(kb),
	)

	if _, err := a.Chat(context.Background(), "what is the return policy?"); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if kb.calls != 1 {
		t.Fatalf("expected 1 Retrieve call, got %d", kb.calls)
	}
	if kb.lastQry != "what is the return policy?" {
		t.Errorf("Retrieve query: got %q", kb.lastQry)
	}

	if mock.CallCount() == 0 {
		t.Fatalf("expected LLM call")
	}

	msgs := mock.calls[0]
	var sysText string
	for _, m := range msgs {
		if m.Role != message.System {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok {
				sysText += tc.Text
			}
		}
	}
	if !strings.Contains(sysText, "Relevant context from the knowledge base") {
		t.Errorf("system prompt missing KB header; got: %q", sysText)
	}
	if !strings.Contains(sysText, "policy-returns") {
		t.Errorf("system prompt missing doc ID; got: %q", sysText)
	}
	if !strings.Contains(sysText, "Returns are accepted within 30 days.") {
		t.Errorf("system prompt missing chunk content; got: %q", sysText)
	}
}

// TestKnowledgeBase_RetrieveErrorDoesNotBlockChat confirms that a
// Retrieve failure is swallowed (so the agent still answers) and the
// system prompt simply lacks KB context.
func TestKnowledgeBase_RetrieveErrorDoesNotBlockChat(t *testing.T) {
	t.Parallel()

	kb := &fakeKB{id: "docs", err: errors.New("boom")}

	mock := newMockLLM(mockResponse{Content: "ok"})
	a := agent.New(mock,
		agent.WithSystemPrompt("you are helpful"),
		agent.WithKnowledgeBase(kb),
	)

	if _, err := a.Chat(context.Background(), "anything"); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if mock.CallCount() != 1 {
		t.Fatalf("expected 1 LLM call, got %d", mock.CallCount())
	}

	msgs := mock.calls[0]
	for _, m := range msgs {
		if m.Role != message.System {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok &&
				strings.Contains(
					tc.Text,
					"Relevant context from the knowledge base",
				) {
				t.Errorf(
					"unexpected KB header on Retrieve error; got: %q",
					tc.Text,
				)
			}
		}
	}
}

// TestKnowledgeBase_NoKBNoEffect is a regression check that the
// option is opt-in and the system prompt looks the same as it would
// without it.
func TestKnowledgeBase_NoKBNoEffect(t *testing.T) {
	t.Parallel()

	mock := newMockLLM(mockResponse{Content: "ok"})
	a := agent.New(mock,
		agent.WithSystemPrompt("you are helpful"),
	)

	if _, err := a.Chat(context.Background(), "hi"); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	for _, m := range mock.calls[0] {
		if m.Role != message.System {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok &&
				strings.Contains(
					tc.Text,
					"Relevant context from the knowledge base",
				) {
				t.Errorf("unexpected KB header without WithKnowledgeBase: %q", tc.Text)
			}
		}
	}
}
