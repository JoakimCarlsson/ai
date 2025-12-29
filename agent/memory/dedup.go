package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
)

// DedupEvent represents the type of deduplication action to take.
type DedupEvent string

const (
	DedupEventAdd    DedupEvent = "ADD"
	DedupEventUpdate DedupEvent = "UPDATE"
	DedupEventDelete DedupEvent = "DELETE"
	DedupEventNone   DedupEvent = "NONE"
)

// DedupDecision represents a single deduplication decision.
type DedupDecision struct {
	Event    DedupEvent `json:"event"`
	MemoryID string     `json:"memory_id,omitempty"`
	Text     string     `json:"text"`
}

// DedupResult contains all deduplication decisions for a fact.
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

// Deduplicate checks if a new fact conflicts with or duplicates existing memories.
// It uses an LLM to decide whether to ADD, UPDATE, DELETE, or skip the new fact.
func Deduplicate(
	ctx context.Context,
	llmClient llm.LLM,
	newFact string,
	existing []Entry,
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
