package agent

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/agent/memory"
)

func (a *Agent) extractAndStoreMemories(ctx context.Context) error {
	if a.memory == nil || !a.autoExtract || a.memoryID == "" ||
		a.session == nil {
		return nil
	}

	messages, err := a.session.GetMessages(ctx, nil)
	if err != nil {
		return err
	}

	facts, err := memory.ExtractFacts(ctx, a.getMemoryLLM(), messages)
	if err != nil {
		return err
	}

	for _, fact := range facts {
		metadata := map[string]any{
			"source":     "auto_extract",
			"created_at": time.Now().Format(time.RFC3339),
		}
		var storeErr error
		if a.autoDedup {
			storeErr = a.storeWithDedup(ctx, fact, metadata)
		} else {
			storeErr = a.memory.Store(ctx, a.memoryID, fact, metadata)
		}
		if storeErr != nil {
			continue
		}
	}

	return nil
}

func (a *Agent) storeWithDedup(
	ctx context.Context,
	fact string,
	metadata map[string]any,
) error {
	if !a.autoDedup || a.memory == nil || a.memoryID == "" {
		return a.memory.Store(ctx, a.memoryID, fact, metadata)
	}

	existing, err := a.memory.Search(ctx, a.memoryID, fact, 5)
	if err != nil {
		return a.memory.Store(ctx, a.memoryID, fact, metadata)
	}

	result, err := memory.Deduplicate(ctx, a.getMemoryLLM(), fact, existing)
	if err != nil {
		return a.memory.Store(ctx, a.memoryID, fact, metadata)
	}

	for _, decision := range result.Decisions {
		switch decision.Event {
		case memory.DedupEventAdd:
			if err := a.memory.Store(ctx, a.memoryID, decision.Text, metadata); err != nil {
				return err
			}
		case memory.DedupEventUpdate:
			if err := a.memory.Update(ctx, decision.MemoryID, decision.Text, metadata); err != nil {
				return err
			}
		case memory.DedupEventDelete:
			if err := a.memory.Delete(ctx, decision.MemoryID); err != nil {
				return err
			}
		case memory.DedupEventNone:
		}
	}

	return nil
}
