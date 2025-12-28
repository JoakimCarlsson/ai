package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

type SimpleMemory struct {
	entries map[string][]agent.MemoryEntry
	mu      sync.RWMutex
	counter int
}

func NewSimpleMemory() *SimpleMemory {
	return &SimpleMemory{
		entries: make(map[string][]agent.MemoryEntry),
	}
}

func (m *SimpleMemory) Store(ctx context.Context, userID string, fact string, metadata map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counter++
	m.entries[userID] = append(m.entries[userID], agent.MemoryEntry{
		ID:        fmt.Sprintf("mem-%d", m.counter),
		Content:   fact,
		UserID:    userID,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	})
	return nil
}

func (m *SimpleMemory) Search(ctx context.Context, userID string, query string, limit int) ([]agent.MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userEntries := m.entries[userID]
	if len(userEntries) == 0 {
		return nil, nil
	}
	if limit > len(userEntries) {
		limit = len(userEntries)
	}
	result := make([]agent.MemoryEntry, limit)
	copy(result, userEntries[len(userEntries)-limit:])
	for i := range result {
		result[i].Score = 0.9
	}
	return result, nil
}

func (m *SimpleMemory) GetAll(ctx context.Context, userID string, limit int) ([]agent.MemoryEntry, error) {
	return m.Search(ctx, userID, "", limit)
}

func (m *SimpleMemory) Delete(ctx context.Context, memoryID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, entries := range m.entries {
		for i, entry := range entries {
			if entry.ID == memoryID {
				m.entries[userID] = append(entries[:i], entries[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("memory not found: %s", memoryID)
}

func main() {
	ctx := context.Background()

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	memory := NewSimpleMemory()

	myAgent := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with memory capabilities. 
Use store_memory when users share personal information or preferences.
Use recall_memories before answering questions that might benefit from user context.
Use delete_memory when users ask you to forget something.`),
		agent.WithMemory(memory),
	)

	ctx = context.WithValue(ctx, "user_id", "alice")

	session, err := agent.NewFileSession("conv-1", "./sessions")
	if err != nil {
		log.Fatal(err)
	}

	response, err := myAgent.Chat(ctx, session, "Hi! My name is Alice and I'm allergic to peanuts.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	session2, err := agent.NewFileSession("conv-2", "./sessions")
	if err != nil {
		log.Fatal(err)
	}

	response, err = myAgent.Chat(ctx, session2, "Can you recommend a restaurant for me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	memories, _ := memory.GetAll(ctx, "alice", 10)
	for _, m := range memories {
		fmt.Printf("[%s] %s\n", m.ID, m.Content)
	}
}
