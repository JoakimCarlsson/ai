// Package memory provides an in-process implementation of rag.Store
// backed by a slice + RWMutex with cosine-similarity scoring. Suitable
// for examples, tests, and small-scale prototypes; data is lost when
// the process exits.
//
// For persistence and scale, use a dedicated rag/store/* implementation
// (e.g., pgvector when it lands).
package memory

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/joakimcarlsson/ai/rag"
)

// New constructs an in-process rag.Store. The store is safe for
// concurrent use across goroutines.
func New() rag.Store {
	return &store{entries: make(map[string][]rag.EmbeddedChunk)}
}

type store struct {
	mu      sync.RWMutex
	entries map[string][]rag.EmbeddedChunk
}

func (s *store) Upsert(
	_ context.Context,
	kbID string,
	chunks []rag.EmbeddedChunk,
) error {
	if len(chunks) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.entries[kbID]

	idx := make(map[string]int, len(existing))
	for i, e := range existing {
		idx[e.ID] = i
	}

	for _, c := range chunks {
		if c.ID == "" {
			existing = append(existing, c)
			continue
		}
		if i, ok := idx[c.ID]; ok {
			existing[i] = c
			continue
		}
		idx[c.ID] = len(existing)
		existing = append(existing, c)
	}

	s.entries[kbID] = existing
	return nil
}

// Search returns the top-k chunks ranked by cosine similarity. Variadic
// SearchOptions are accepted for forward compatibility but currently
// have no effect (the v1 store does not implement filters or score
// thresholds; KnowledgeBase-level WithMinScore / WithMaxDistance handle
// thresholds at the orchestrator layer).
func (s *store) Search(
	_ context.Context,
	kbID string,
	query []float32,
	k int,
	opts ...rag.SearchOption,
) ([]rag.Hit, error) {
	if k <= 0 || len(query) == 0 {
		return nil, nil
	}

	// Drain options for forward-compat. v1 has no readable fields on
	// SearchConfig, but invoking the helper validates that callers do
	// not pass nil options and exercises the option pipeline so future
	// additions are guaranteed to be wired through every store.
	_ = rag.ApplySearchOptions(opts...)

	s.mu.RLock()
	chunks := s.entries[kbID]
	s.mu.RUnlock()

	if len(chunks) == 0 {
		return nil, nil
	}

	scored := make([]rag.Hit, 0, len(chunks))
	for _, c := range chunks {
		score := cosineSimilarity(query, c.Embedding)
		scored = append(scored, rag.Hit{Chunk: c.Chunk, Score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if k > len(scored) {
		k = len(scored)
	}
	return scored[:k], nil
}

func (s *store) Delete(_ context.Context, chunkID string) error {
	if chunkID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for kbID, chunks := range s.entries {
		for i, c := range chunks {
			if c.ID == chunkID {
				s.entries[kbID] = append(chunks[:i], chunks[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
