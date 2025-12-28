package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

type memoryWithVector struct {
	entry  agent.MemoryEntry
	vector []float32
}

type VectorMemory struct {
	embedder embeddings.Embedding
	entries  map[string][]memoryWithVector
	mu       sync.RWMutex
	counter  int
}

func NewVectorMemory(embedder embeddings.Embedding) *VectorMemory {
	return &VectorMemory{
		embedder: embedder,
		entries:  make(map[string][]memoryWithVector),
	}
}

func (m *VectorMemory) Store(ctx context.Context, userID string, fact string, metadata map[string]any) error {
	resp, err := m.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.counter++
	m.entries[userID] = append(m.entries[userID], memoryWithVector{
		entry: agent.MemoryEntry{
			ID:        fmt.Sprintf("mem-%d", m.counter),
			Content:   fact,
			UserID:    userID,
			CreatedAt: time.Now(),
			Metadata:  metadata,
		},
		vector: resp.Embeddings[0],
	})
	return nil
}

func (m *VectorMemory) Search(ctx context.Context, userID string, query string, limit int) ([]agent.MemoryEntry, error) {
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
		score := cosineSimilarity(queryVector, mem.vector)
		entry := mem.entry
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
		out[i] = userEntries[i].entry
	}
	return out, nil
}

func (m *VectorMemory) Delete(ctx context.Context, memoryID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, entries := range m.entries {
		for i, mem := range entries {
			if mem.entry.ID == memoryID {
				m.entries[userID] = append(entries[:i], entries[i+1:]...)
				return nil
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

	memory := NewVectorMemory(embedder)

	myAgent := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with semantic memory.
Use store_memory when users share personal information or preferences.
Use recall_memories to find relevant context before answering questions.`),
		agent.WithMemory(memory),
	)

	ctx = context.WithValue(ctx, "user_id", "alice")

	session, err := agent.NewFileSession("conv-1", "./sessions")
	if err != nil {
		log.Fatal(err)
	}

	response, err := myAgent.Chat(ctx, session, "I love hiking in the mountains and prefer vegetarian food.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	session2, err := agent.NewFileSession("conv-2", "./sessions")
	if err != nil {
		log.Fatal(err)
	}

	response, err = myAgent.Chat(ctx, session2, "What outdoor activities would you recommend for me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	memories, _ := memory.Search(ctx, "alice", "food preferences", 5)
	for _, m := range memories {
		fmt.Printf("[%.2f] %s\n", m.Score, m.Content)
	}
}
