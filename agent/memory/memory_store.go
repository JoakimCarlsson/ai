package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/embeddings"
)

// storedEntry holds a memory entry along with its vector embedding.
type storedEntry struct {
	Entry
	Vector []float32 `json:"vector"`
}

// memoryStore is an in-memory implementation of Store.
// Data is lost when the process exits.
type memoryStore struct {
	embedder embeddings.Embedding
	entries  map[string][]storedEntry // keyed by ownerID
	mu       sync.RWMutex
}

// MemoryStore creates an in-memory Store that uses the provided embedder
// for vector similarity search. Data is not persisted and will be lost
// when the process exits.
func MemoryStore(embedder embeddings.Embedding) Store {
	return &memoryStore{
		embedder: embedder,
		entries:  make(map[string][]storedEntry),
	}
}

func (s *memoryStore) Store(ctx context.Context, id string, fact string, metadata map[string]any) error {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}

	entry := storedEntry{
		Entry: Entry{
			ID:        uuid.New().String(),
			Content:   fact,
			OwnerID:   id,
			CreatedAt: time.Now(),
			Metadata:  metadata,
		},
		Vector: resp.Embeddings[0],
	}

	s.mu.Lock()
	s.entries[id] = append(s.entries[id], entry)
	s.mu.Unlock()

	return nil
}

func (s *memoryStore) Search(ctx context.Context, id string, query string, limit int) ([]Entry, error) {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVector := resp.Embeddings[0]

	s.mu.RLock()
	userEntries := s.entries[id]
	s.mu.RUnlock()

	if len(userEntries) == 0 {
		return []Entry{}, nil
	}

	type scored struct {
		entry storedEntry
		score float64
	}

	scoredEntries := make([]scored, len(userEntries))
	for i, e := range userEntries {
		scoredEntries[i] = scored{
			entry: e,
			score: cosineSimilarity(queryVector, e.Vector),
		}
	}

	sort.Slice(scoredEntries, func(i, j int) bool {
		return scoredEntries[i].score > scoredEntries[j].score
	})

	if limit > len(scoredEntries) {
		limit = len(scoredEntries)
	}

	results := make([]Entry, limit)
	for i := 0; i < limit; i++ {
		results[i] = scoredEntries[i].entry.Entry
		results[i].Score = scoredEntries[i].score
	}

	return results, nil
}

func (s *memoryStore) GetAll(ctx context.Context, id string, limit int) ([]Entry, error) {
	s.mu.RLock()
	userEntries := s.entries[id]
	s.mu.RUnlock()

	if limit > len(userEntries) {
		limit = len(userEntries)
	}

	results := make([]Entry, limit)
	for i := 0; i < limit; i++ {
		results[i] = userEntries[i].Entry
	}

	return results, nil
}

func (s *memoryStore) Delete(ctx context.Context, memoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for ownerID, entries := range s.entries {
		for i, e := range entries {
			if e.ID == memoryID {
				s.entries[ownerID] = append(entries[:i], entries[i+1:]...)
				return nil
			}
		}
	}

	return nil
}

func (s *memoryStore) Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}
	newVector := resp.Embeddings[0]

	s.mu.Lock()
	defer s.mu.Unlock()

	for ownerID, entries := range s.entries {
		for i, e := range entries {
			if e.ID == memoryID {
				s.entries[ownerID][i].Content = fact
				s.entries[ownerID][i].Vector = newVector
				if metadata != nil {
					s.entries[ownerID][i].Metadata = metadata
				}
				return nil
			}
		}
	}

	return nil
}

