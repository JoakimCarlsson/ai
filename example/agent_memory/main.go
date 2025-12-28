package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

type FileMemory struct {
	dir     string
	entries map[string][]agent.MemoryEntry
	mu      sync.RWMutex
	counter int
}

func NewFileMemory(dir string) (*FileMemory, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	m := &FileMemory{
		dir:     dir,
		entries: make(map[string][]agent.MemoryEntry),
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range files {
		userID := filepath.Base(f[:len(f)-5])
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var entries []agent.MemoryEntry
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

func (m *FileMemory) save(userID string) error {
	data, err := json.MarshalIndent(m.entries[userID], "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.dir, userID+".json"), data, 0644)
}

func (m *FileMemory) Store(ctx context.Context, userID string, fact string, metadata map[string]any) error {
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
	return m.save(userID)
}

func (m *FileMemory) Search(ctx context.Context, userID string, query string, limit int) ([]agent.MemoryEntry, error) {
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

func (m *FileMemory) GetAll(ctx context.Context, userID string, limit int) ([]agent.MemoryEntry, error) {
	return m.Search(ctx, userID, "", limit)
}

func (m *FileMemory) Delete(ctx context.Context, memoryID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, entries := range m.entries {
		for i, entry := range entries {
			if entry.ID == memoryID {
				m.entries[userID] = append(entries[:i], entries[i+1:]...)
				return m.save(userID)
			}
		}
	}
	return fmt.Errorf("memory not found: %s", memoryID)
}

func (m *FileMemory) Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, entries := range m.entries {
		for i, entry := range entries {
			if entry.ID == memoryID {
				m.entries[userID][i].Content = fact
				m.entries[userID][i].Metadata = metadata
				return m.save(userID)
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

	memory, err := NewFileMemory("./memories")
	if err != nil {
		log.Fatal(err)
	}

	myAgent := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with memory capabilities. 
Use store_memory when users share personal information or preferences.
Use recall_memories before answering questions that might benefit from user context.
Use replace_memory when information has changed (first recall to get the memory_id).
Use delete_memory when users ask you to forget something.`),
		agent.WithMemory(memory),
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

	response, err := myAgent.Chat(ctx, session, "Hi! My name is Alice and I'm allergic to peanuts.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	session2, err := agent.GetOrCreateSession(ctx, "conv-2", store)
	if err != nil {
		log.Fatal(err)
	}

	response, err = myAgent.Chat(ctx, session2, "Can you recommend a restaurant for me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
