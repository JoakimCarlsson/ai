package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/embeddings"
)

// fileStore is a file-based implementation of Store.
// Each owner's memories are stored in a separate JSON file.
type fileStore struct {
	dir      string
	embedder embeddings.Embedding
	mu       sync.RWMutex
}

// FileStore creates a file-based Store that persists memories to disk.
// Each owner's memories are stored in a separate JSON file in the specified directory.
// The embedder is used for vector similarity search.
func FileStore(dir string, embedder embeddings.Embedding) Store {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil
	}
	return &fileStore{
		dir:      dir,
		embedder: embedder,
	}
}

func (s *fileStore) filePath(ownerID string) string {
	return filepath.Join(s.dir, ownerID+".json")
}

func (s *fileStore) loadEntries(ownerID string) ([]storedEntry, error) {
	data, err := os.ReadFile(s.filePath(ownerID))
	if err != nil {
		if os.IsNotExist(err) {
			return []storedEntry{}, nil
		}
		return nil, err
	}

	var entries []storedEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *fileStore) saveEntries(ownerID string, entries []storedEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath(ownerID), data, 0644)
}

func (s *fileStore) Store(ctx context.Context, id string, fact string, metadata map[string]any) error {
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
	defer s.mu.Unlock()

	entries, err := s.loadEntries(id)
	if err != nil {
		return err
	}

	entries = append(entries, entry)
	return s.saveEntries(id, entries)
}

func (s *fileStore) Search(ctx context.Context, id string, query string, limit int) ([]Entry, error) {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVector := resp.Embeddings[0]

	s.mu.RLock()
	entries, err := s.loadEntries(id)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []Entry{}, nil
	}

	type scored struct {
		entry storedEntry
		score float64
	}

	scoredEntries := make([]scored, len(entries))
	for i, e := range entries {
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

func (s *fileStore) GetAll(ctx context.Context, id string, limit int) ([]Entry, error) {
	s.mu.RLock()
	entries, err := s.loadEntries(id)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	if limit > len(entries) {
		limit = len(entries)
	}

	results := make([]Entry, limit)
	for i := 0; i < limit; i++ {
		results[i] = entries[i].Entry
	}

	return results, nil
}

func (s *fileStore) Delete(ctx context.Context, memoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We need to search all files since we don't know which owner this memory belongs to
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		ownerID := file.Name()[:len(file.Name())-5] // remove .json
		entries, err := s.loadEntries(ownerID)
		if err != nil {
			continue
		}

		for i, e := range entries {
			if e.ID == memoryID {
				entries = append(entries[:i], entries[i+1:]...)
				return s.saveEntries(ownerID, entries)
			}
		}
	}

	return nil
}

func (s *fileStore) Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error {
	resp, err := s.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}
	newVector := resp.Embeddings[0]

	s.mu.Lock()
	defer s.mu.Unlock()

	// We need to search all files since we don't know which owner this memory belongs to
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		ownerID := file.Name()[:len(file.Name())-5] // remove .json
		entries, err := s.loadEntries(ownerID)
		if err != nil {
			continue
		}

		for i, e := range entries {
			if e.ID == memoryID {
				entries[i].Content = fact
				entries[i].Vector = newVector
				if metadata != nil {
					entries[i].Metadata = metadata
				}
				return s.saveEntries(ownerID, entries)
			}
		}
	}

	return nil
}

