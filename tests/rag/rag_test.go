package rag

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/rag"
	ragmem "github.com/joakimcarlsson/ai/rag/store/memory"
	"github.com/joakimcarlsson/ai/tool"
)

func TestKnowledgeBaseIngestAndRetrieveRoundtrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	embedder := newFakeEmbedder()
	store := ragmem.New()
	kb := rag.New("docs", embedder, store)

	docs := []rag.Document{
		{
			ID:      "faq-billing",
			Content: "Customers can update their billing email in the account settings page under Billing.",
		},
		{
			ID:      "faq-shipping",
			Content: "Standard shipping takes 3 to 5 business days. Express shipping arrives next day.",
		},
		{
			ID:      "faq-returns",
			Content: "Items can be returned within 30 days of purchase for a full refund.",
		},
	}

	if err := kb.Ingest(ctx, docs); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	hits, err := kb.Retrieve(ctx, "How do I change my billing email?", 3)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected hits, got 0")
	}
	if hits[0].DocumentID != "faq-billing" {
		t.Errorf(
			"top hit: want faq-billing, got %s",
			hits[0].DocumentID,
		)
	}
}

func TestKnowledgeBaseRetrieveAppliesMinScore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	embedder := newFakeEmbedder()
	store := ragmem.New()
	kb := rag.New("docs", embedder, store, rag.WithMinScore(0.99))

	docs := []rag.Document{
		{ID: "a", Content: "alpha bravo"},
		{ID: "b", Content: "charlie delta"},
	}
	if err := kb.Ingest(ctx, docs); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	hits, err := kb.Retrieve(ctx, "echo foxtrot", 5)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf(
			"expected 0 hits with min-score 0.99 on disjoint query, got %d",
			len(hits),
		)
	}
}

func TestKnowledgeBaseRetrieveEmptyInputs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	kb := rag.New("docs", newFakeEmbedder(), ragmem.New())

	hits, err := kb.Retrieve(ctx, "", 5)
	if err != nil {
		t.Fatalf("Retrieve(\"\"): %v", err)
	}
	if hits != nil {
		t.Errorf(
			"Retrieve with empty query should return nil, got %d hits",
			len(hits),
		)
	}

	hits, err = kb.Retrieve(ctx, "anything", 0)
	if err != nil {
		t.Fatalf("Retrieve(k=0): %v", err)
	}
	if hits != nil {
		t.Errorf("Retrieve with k=0 should return nil, got %d hits", len(hits))
	}
}

func TestKnowledgeBaseIngestStampsEmbeddingModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	embedder := newFakeEmbedder()
	store := &recordingStore{inner: ragmem.New()}
	kb := rag.New("docs", embedder, store)

	if err := kb.Ingest(ctx, []rag.Document{
		{ID: "d1", Content: "alpha bravo charlie"},
	}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	if len(store.upserts) == 0 {
		t.Fatalf("expected at least one Upsert call")
	}
	for _, batch := range store.upserts {
		for _, c := range batch {
			if c.Model != "fake-embed-001" {
				t.Errorf(
					"chunk %s: Model = %q, want %q",
					c.ID, c.Model, "fake-embed-001",
				)
			}
		}
	}
}

func TestSearchToolWiring(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	embedder := newFakeEmbedder()
	store := ragmem.New()
	kb := rag.New("docs", embedder, store)

	if err := kb.Ingest(ctx, []rag.Document{
		{
			ID:      "policy-returns",
			Content: "Returns are accepted within 30 days for a full refund.",
		},
		{
			ID:      "policy-shipping",
			Content: "Shipping takes 3 to 5 days domestically.",
		},
	}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	st := rag.SearchTool(kb)

	info := st.Info()
	if info.Name != "search_knowledge_base" {
		t.Errorf("Name: got %q, want search_knowledge_base", info.Name)
	}
	if !contains(info.Required, "query") {
		t.Errorf("Required: %v missing 'query'", info.Required)
	}

	resp, err := st.Run(ctx, tool.Call{
		ID:    "c1",
		Name:  info.Name,
		Input: `{"query": "how long do I have to return", "k": 2}`,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.IsError {
		t.Fatalf("unexpected tool error: %s", resp.Content)
	}
	if !strings.Contains(resp.Content, "policy-returns") {
		t.Errorf(
			"tool response missing expected doc id; got: %s",
			resp.Content,
		)
	}
}

func TestSearchToolEmptyQueryRejected(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	kb := rag.New("docs", newFakeEmbedder(), ragmem.New())
	st := rag.SearchTool(kb)

	resp, err := st.Run(ctx, tool.Call{
		ID:    "c1",
		Name:  st.Info().Name,
		Input: `{"query": ""}`,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !resp.IsError {
		t.Errorf("expected IsError on empty query, got %s", resp.Content)
	}
}

func TestFormatHits(t *testing.T) {
	t.Parallel()

	got := rag.FormatHits([]rag.Hit{
		{
			Chunk: rag.Chunk{DocumentID: "doc-1", Content: "  alpha\n"},
			Score: 0.9,
		},
		{
			Chunk: rag.Chunk{DocumentID: "doc-2", Content: "bravo"},
			Score: 0.8,
		},
	})
	want := "1. [doc-1] alpha\n2. [doc-2] bravo"
	if got != want {
		t.Errorf("FormatHits:\ngot:\n%s\nwant:\n%s", got, want)
	}

	if got := rag.FormatHits(nil); !strings.Contains(got, "No relevant") {
		t.Errorf("FormatHits(nil) = %q, want sentinel string", got)
	}
}

func contains(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}

// recordingStore wraps another rag.Store and captures Upsert batches
// so tests can assert on EmbeddedChunk fields populated by the
// KnowledgeBase orchestrator.
type recordingStore struct {
	inner   rag.Store
	upserts [][]rag.EmbeddedChunk
}

func (s *recordingStore) Upsert(
	ctx context.Context,
	kbID string,
	chunks []rag.EmbeddedChunk,
) error {
	s.upserts = append(s.upserts, append(
		[]rag.EmbeddedChunk(nil), chunks...,
	))
	return s.inner.Upsert(ctx, kbID, chunks)
}

func (s *recordingStore) Search(
	ctx context.Context,
	kbID string,
	embedding []float32,
	k int,
	opts ...rag.SearchOption,
) ([]rag.Hit, error) {
	return s.inner.Search(ctx, kbID, embedding, k, opts...)
}

func (s *recordingStore) Delete(ctx context.Context, chunkID string) error {
	return s.inner.Delete(ctx, chunkID)
}
