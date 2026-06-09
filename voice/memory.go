package voice

import (
	"context"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/memory"
	"github.com/joakimcarlsson/ai/message"
)

// getMemoryLLM returns the LLM used for memory extraction and dedup. Uses
// the dedicated memory LLM if configured, falling back to the agent's
// main LLM.
func (v *Agent) getMemoryLLM() llm.LLM {
	if v.memoryLLM != nil {
		return v.memoryLLM
	}
	return v.llm
}

// extractAndStoreMemories pulls the session's full message history,
// extracts facts via memory.ExtractFacts, and stores each (with dedup if
// configured). Mirrors agent.extractAndStoreMemories. Intended to be
// invoked in a background goroutine after each user turn ends — the runner
// fires it with context.Background() so an extraction outlives the
// conversation cancellation.
func (v *Agent) extractAndStoreMemories(ctx context.Context) error {
	if v.memory == nil || !v.autoExtract || v.memoryID == "" ||
		v.session == nil {
		return nil
	}

	messages, err := v.session.GetMessages(ctx, nil)
	if err != nil {
		return err
	}

	facts, err := memory.ExtractFacts(ctx, v.getMemoryLLM(), messages)
	if err != nil {
		return err
	}

	for _, fact := range facts {
		metadata := map[string]any{
			"source":     "auto_extract",
			"created_at": time.Now().Format(time.RFC3339),
		}
		if v.autoDedup {
			_ = v.storeWithDedup(ctx, fact, metadata)
		} else {
			_ = v.memory.Store(ctx, v.memoryID, fact, metadata)
		}
	}
	return nil
}

// storeWithDedup runs memory.Deduplicate against the top-5 nearest
// memories before storing the new fact. Apply each Add/Update/Delete
// decision; on any dedup error, fall back to a plain Store.
func (v *Agent) storeWithDedup(
	ctx context.Context,
	fact string,
	metadata map[string]any,
) error {
	if !v.autoDedup || v.memory == nil || v.memoryID == "" {
		return v.memory.Store(ctx, v.memoryID, fact, metadata)
	}

	existing, err := v.memory.Search(ctx, v.memoryID, fact, 5)
	if err != nil {
		return v.memory.Store(ctx, v.memoryID, fact, metadata)
	}

	result, err := memory.Deduplicate(ctx, v.getMemoryLLM(), fact, existing)
	if err != nil {
		return v.memory.Store(ctx, v.memoryID, fact, metadata)
	}

	for _, decision := range result.Decisions {
		switch decision.Event {
		case memory.DedupEventAdd:
			if err := v.memory.Store(
				ctx,
				v.memoryID,
				decision.Text,
				metadata,
			); err != nil {
				return err
			}
		case memory.DedupEventUpdate:
			if err := v.memory.Update(
				ctx,
				decision.MemoryID,
				decision.Text,
				metadata,
			); err != nil {
				return err
			}
		case memory.DedupEventDelete:
			if err := v.memory.Delete(ctx, decision.MemoryID); err != nil {
				return err
			}
		case memory.DedupEventNone:
		}
	}
	return nil
}

// recallMemoriesContext searches the configured memory store for the
// top-N memories matching query and returns them as a single
// system-message-shaped string ready to be prepended to the LLM message
// list. Returns "" when no memories are configured or none match.
func (v *Agent) recallMemoriesContext(
	ctx context.Context,
	query string,
	limit int,
) string {
	if v.memory == nil || v.memoryID == "" || strings.TrimSpace(query) == "" {
		return ""
	}
	hits, err := v.memory.Search(ctx, v.memoryID, query, limit)
	if err != nil || len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Relevant memories about this user:\n")
	for _, m := range hits {
		b.WriteString("- ")
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
}

// lastUserText returns the text of the most recent message.User in
// history, or "" if none. Used as the recall query.
func lastUserText(history []message.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.Role != message.User {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok {
				return tc.Text
			}
		}
	}
	return ""
}
