package rag

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/rag"
	ragmem "github.com/joakimcarlsson/ai/rag/store/memory"
)

func TestMemoryStoreUpsertAndSearch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := ragmem.New()
	chunks := []rag.EmbeddedChunk{
		{
			Chunk: rag.Chunk{
				ID:         "c1",
				DocumentID: "d1",
				Content:    "alpha bravo charlie",
			},
			Embedding: []float32{1, 0, 0},
			Model:     "test-embed-001",
		},
		{
			Chunk: rag.Chunk{
				ID:         "c2",
				DocumentID: "d1",
				Content:    "delta echo",
			},
			Embedding: []float32{0, 1, 0},
			Model:     "test-embed-001",
		},
	}
	if err := store.Upsert(ctx, "kb-1", chunks); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	hits, err := store.Search(ctx, "kb-1", []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if hits[0].ID != "c1" {
		t.Errorf("top hit: want c1, got %s", hits[0].ID)
	}
	if hits[0].Score < hits[1].Score {
		t.Errorf("hits not ordered by score: %v", hits)
	}
}

func TestMemoryStoreUpsertOverwritesByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := ragmem.New()
	if err := store.Upsert(ctx, "kb", []rag.EmbeddedChunk{{
		Chunk:     rag.Chunk{ID: "c1", Content: "v1"},
		Embedding: []float32{1, 0},
	}}); err != nil {
		t.Fatalf("Upsert v1: %v", err)
	}
	if err := store.Upsert(ctx, "kb", []rag.EmbeddedChunk{{
		Chunk:     rag.Chunk{ID: "c1", Content: "v2"},
		Embedding: []float32{1, 0},
	}}); err != nil {
		t.Fatalf("Upsert v2: %v", err)
	}

	hits, err := store.Search(ctx, "kb", []float32{1, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Errorf("expected 1 chunk after overwrite, got %d", len(hits))
	}
	if hits[0].Content != "v2" {
		t.Errorf("Content: got %q, want v2", hits[0].Content)
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := ragmem.New()
	if err := store.Upsert(ctx, "kb", []rag.EmbeddedChunk{
		{
			Chunk:     rag.Chunk{ID: "c1", Content: "a"},
			Embedding: []float32{1, 0},
		},
		{
			Chunk:     rag.Chunk{ID: "c2", Content: "b"},
			Embedding: []float32{0, 1},
		},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := store.Delete(ctx, "c1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	hits, err := store.Search(ctx, "kb", []float32{1, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, h := range hits {
		if h.ID == "c1" {
			t.Errorf("c1 still present after Delete")
		}
	}

	if err := store.Delete(ctx, "does-not-exist"); err != nil {
		t.Errorf("Delete missing: want nil, got %v", err)
	}
}

func TestMemoryStoreSearchAcceptsAndIgnoresOptions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := ragmem.New()
	if err := store.Upsert(ctx, "kb", []rag.EmbeddedChunk{{
		Chunk:     rag.Chunk{ID: "c1", Content: "a"},
		Embedding: []float32{1, 0},
	}}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// v1 ships no exported SearchOption helpers; we exercise the
	// variadic path by passing zero options. The point is forward
	// compatibility: store implementations must not crash when called
	// without options, and adding options later must not break this
	// call shape.
	hits, err := store.Search(ctx, "kb", []float32{1, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Errorf("expected 1 hit, got %d", len(hits))
	}
}

func TestMemoryStoreSearchHandlesEmptyKB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := ragmem.New()
	hits, err := store.Search(ctx, "kb", []float32{1, 0}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if hits != nil {
		t.Errorf("expected nil hits on empty store, got %d", len(hits))
	}
}
