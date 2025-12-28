package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

type storedMemory struct {
	Entry  agent.MemoryEntry `json:"entry"`
	Vector []float32         `json:"vector"`
}

type VectorMemory struct {
	dir      string
	embedder embeddings.Embedding
	entries  map[string][]storedMemory
	mu       sync.RWMutex
	counter  int
}

func NewVectorMemory(dir string, embedder embeddings.Embedding) (*VectorMemory, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	m := &VectorMemory{
		dir:      dir,
		embedder: embedder,
		entries:  make(map[string][]storedMemory),
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range files {
		userID := filepath.Base(f[:len(f)-5])
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var entries []storedMemory
		if err := json.Unmarshal(data, &entries); err != nil {
			continue
		}
		m.entries[userID] = entries
		if len(entries) > m.counter {
			m.counter = len(entries)
		}
	}

	return m, nil
}

func (m *VectorMemory) save(userID string) error {
	data, err := json.MarshalIndent(m.entries[userID], "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.dir, userID+".json"), data, 0644)
}

func (m *VectorMemory) Store(ctx context.Context, userID string, fact string, metadata map[string]any) error {
	resp, err := m.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.counter++
	m.entries[userID] = append(m.entries[userID], storedMemory{
		Entry: agent.MemoryEntry{
			ID:        fmt.Sprintf("mem-%d", m.counter),
			Content:   fact,
			UserID:    userID,
			CreatedAt: time.Now(),
			Metadata:  metadata,
		},
		Vector: resp.Embeddings[0],
	})
	return m.save(userID)
}

func (m *VectorMemory) Search(
	ctx context.Context,
	userID string,
	query string,
	limit int,
) ([]agent.MemoryEntry, error) {
	resp, err := m.embedder.GenerateEmbeddings(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVector := resp.Embeddings[0]

	m.mu.RLock()
	defer m.mu.RUnlock()

	userEntries := m.entries[userID]
	if len(userEntries) == 0 {
		return nil, nil
	}

	type scored struct {
		entry agent.MemoryEntry
		score float64
	}
	var results []scored

	for _, mem := range userEntries {
		score := cosineSimilarity(queryVector, mem.Vector)
		entry := mem.Entry
		entry.Score = score
		results = append(results, scored{entry: entry, score: score})
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit > len(results) {
		limit = len(results)
	}

	out := make([]agent.MemoryEntry, limit)
	for i := 0; i < limit; i++ {
		out[i] = results[i].entry
	}
	return out, nil
}

func (m *VectorMemory) GetAll(ctx context.Context, userID string, limit int) ([]agent.MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userEntries := m.entries[userID]
	if limit > len(userEntries) {
		limit = len(userEntries)
	}

	out := make([]agent.MemoryEntry, limit)
	for i := 0; i < limit; i++ {
		out[i] = userEntries[i].Entry
	}
	return out, nil
}

func (m *VectorMemory) Delete(ctx context.Context, memoryID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, entries := range m.entries {
		for i, mem := range entries {
			if mem.Entry.ID == memoryID {
				m.entries[userID] = append(entries[:i], entries[i+1:]...)
				return m.save(userID)
			}
		}
	}
	return fmt.Errorf("memory not found: %s", memoryID)
}

func (m *VectorMemory) Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error {
	resp, err := m.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, entries := range m.entries {
		for i, mem := range entries {
			if mem.Entry.ID == memoryID {
				m.entries[userID][i].Entry.Content = fact
				m.entries[userID][i].Entry.Metadata = metadata
				m.entries[userID][i].Vector = resp.Embeddings[0]
				return m.save(userID)
			}
		}
	}
	return fmt.Errorf("memory not found: %s", memoryID)
}

func cosineSimilarity(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func main() {
	ctx := context.Background()

	embedder, err := embeddings.NewEmbedding(model.ProviderOpenAI,
		embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
	)
	if err != nil {
		log.Fatal(err)
	}

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	memory, err := NewVectorMemory("./memories", embedder)
	if err != nil {
		log.Fatal(err)
	}

	myAgent := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with semantic memory.
Use store_memory when users share personal information or preferences.
Use recall_memories to find relevant context before answering questions.
Use replace_memory when information has changed (first recall to get the memory_id).
Use delete_memory when users ask you to forget something.`),
		agent.WithMemory(memory),
		agent.WithAutoDedup(true),
	)

	ctx = context.WithValue(ctx, "user_id", "alice")

	store, err := agent.NewFileSessionStore("./sessions")
	if err != nil {
		log.Fatal(err)
	}

	session, err := agent.GetOrCreateSession(ctx, "conv-1", store)
	if err != nil {
		log.Fatal(err)
	}

	response, err := myAgent.Chat(ctx, session, "I love hiking in the mountains and prefer vegetarian food.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	session2, err := agent.GetOrCreateSession(ctx, "conv-2", store)
	if err != nil {
		log.Fatal(err)
	}

	response, err = myAgent.Chat(ctx, session2, "What outdoor activities would you recommend for me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
