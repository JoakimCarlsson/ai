package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
)

type DedupEvent string

const (
	DedupEventAdd    DedupEvent = "ADD"
	DedupEventUpdate DedupEvent = "UPDATE"
	DedupEventDelete DedupEvent = "DELETE"
	DedupEventNone   DedupEvent = "NONE"
)

type DedupDecision struct {
	Event    DedupEvent `json:"event"`
	MemoryID string     `json:"memory_id,omitempty"`
	Text     string     `json:"text"`
}

type DedupResult struct {
	Decisions []DedupDecision `json:"decisions"`
}

const dedupSystemPrompt = `You are a memory deduplication assistant. Given existing memories and a new fact, decide what action to take.

For each decision, respond with one of:
- ADD: The new fact is genuinely new information, no existing memory covers it
- UPDATE: An existing memory should be updated with new information (provide memory_id and the new combined text)
- DELETE: An existing memory is now contradicted or completely outdated (provide memory_id)
- NONE: The fact is already covered by existing memories, no action needed

Respond ONLY with valid JSON in this exact format:
{"decisions": [{"event": "ADD|UPDATE|DELETE|NONE", "memory_id": "id if UPDATE or DELETE", "text": "the fact text"}]}

Rules:
1. Prefer UPDATE over DELETE+ADD when information evolves
2. Use DELETE only when information is explicitly contradicted
3. Use NONE when the new fact adds no new information
4. The "text" field should contain the final fact to store (for ADD/UPDATE) or the original fact (for DELETE/NONE)`

func deduplicateMemory(
	ctx context.Context,
	llmClient llm.LLM,
	newFact string,
	existing []MemoryEntry,
) (*DedupResult, error) {
	if len(existing) == 0 {
		return &DedupResult{
			Decisions: []DedupDecision{{
				Event: DedupEventAdd,
				Text:  newFact,
			}},
		}, nil
	}

	var existingStr string
	for _, m := range existing {
		existingStr += fmt.Sprintf("- [id:%s] %s\n", m.ID, m.Content)
	}

	userPrompt := fmt.Sprintf("Existing memories:\n%s\nNew fact to process: %s", existingStr, newFact)

	messages := []message.Message{
		message.NewSystemMessage(dedupSystemPrompt),
		message.NewUserMessage(userPrompt),
	}

	resp, err := llmClient.SendMessages(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("dedup LLM call failed: %w", err)
	}

	var result DedupResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return &DedupResult{
			Decisions: []DedupDecision{{
				Event: DedupEventAdd,
				Text:  newFact,
			}},
		}, nil
	}

	return &result, nil
}

func (a *Agent) storeWithDedup(ctx context.Context, userID string, fact string, metadata map[string]any) error {
	if !a.autoDedup || a.memory == nil {
		return a.memory.Store(ctx, userID, fact, metadata)
	}

	existing, err := a.memory.Search(ctx, userID, fact, 5)
	if err != nil {
		return a.memory.Store(ctx, userID, fact, metadata)
	}

	result, err := deduplicateMemory(ctx, a.getMemoryLLM(), fact, existing)
	if err != nil {
		return a.memory.Store(ctx, userID, fact, metadata)
	}

	for _, decision := range result.Decisions {
		switch decision.Event {
		case DedupEventAdd:
			if err := a.memory.Store(ctx, userID, decision.Text, metadata); err != nil {
				return err
			}
		case DedupEventUpdate:
			if err := a.memory.Update(ctx, decision.MemoryID, decision.Text, metadata); err != nil {
				return err
			}
		case DedupEventDelete:
			if err := a.memory.Delete(ctx, decision.MemoryID); err != nil {
				return err
			}
		case DedupEventNone:
		}
	}

	return nil
}
