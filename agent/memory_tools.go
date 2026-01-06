package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/tool"
)

type storeMemoryTool struct {
	store    memory.Store
	memoryID string
}

func newStoreMemoryTool(store memory.Store, memoryID string) *storeMemoryTool {
	return &storeMemoryTool{store: store, memoryID: memoryID}
}

func (t *storeMemoryTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "store_memory",
		Description: "Store an important fact about the user for future conversations. Use when user shares preferences, personal details, health info, or anything worth remembering long-term.",
		Parameters: map[string]any{
			"fact": map[string]any{
				"type":        "string",
				"description": "The fact to remember about the user",
			},
			"category": map[string]any{
				"type":        "string",
				"enum":        []string{"preference", "personal", "health", "professional", "other"},
				"description": "Category of the memory",
			},
		},
		Required: []string{"fact"},
	}
}

func (t *storeMemoryTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input struct {
		Fact     string `json:"fact"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	metadata := map[string]any{}
	if input.Category != "" {
		metadata["category"] = input.Category
	}

	if err := t.store.Store(ctx, t.memoryID, input.Fact, metadata); err != nil {
		return tool.NewTextErrorResponse("failed to store memory: " + err.Error()), nil
	}

	return tool.NewTextResponse("Memory stored successfully"), nil
}

type recallMemoriesTool struct {
	store    memory.Store
	memoryID string
}

func newRecallMemoriesTool(store memory.Store, memoryID string) *recallMemoriesTool {
	return &recallMemoriesTool{store: store, memoryID: memoryID}
}

func (t *recallMemoriesTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "recall_memories",
		Description: "Search for relevant memories about the user. Use before answering questions that might benefit from knowing user preferences or history.",
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "What to search for in memories",
			},
		},
		Required: []string{"query"},
	}
}

func (t *recallMemoriesTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	memories, err := t.store.Search(ctx, t.memoryID, input.Query, 5)
	if err != nil {
		return tool.NewTextErrorResponse("failed to search memories: " + err.Error()), nil
	}

	if len(memories) == 0 {
		return tool.NewTextResponse("No relevant memories found"), nil
	}

	var results []string
	for _, m := range memories {
		results = append(results, fmt.Sprintf("- [id:%s] %s", m.ID, m.Content))
	}

	return tool.NewTextResponse(strings.Join(results, "\n")), nil
}

type deleteMemoryTool struct {
	store    memory.Store
	memoryID string
}

func newDeleteMemoryTool(store memory.Store, memoryID string) *deleteMemoryTool {
	return &deleteMemoryTool{store: store, memoryID: memoryID}
}

func (t *deleteMemoryTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "delete_memory",
		Description: "Delete a stored memory. Use when the user explicitly asks to forget something, or when information is no longer accurate/relevant.",
		Parameters: map[string]any{
			"memory_id": map[string]any{
				"type":        "string",
				"description": "The ID of the memory to delete (from recall_memories results)",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Why the memory is being deleted",
			},
		},
		Required: []string{"memory_id"},
	}
}

func (t *deleteMemoryTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input struct {
		MemoryID string `json:"memory_id"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	if err := t.store.Delete(ctx, input.MemoryID); err != nil {
		return tool.NewTextErrorResponse("failed to delete memory: " + err.Error()), nil
	}

	return tool.NewTextResponse("Memory deleted successfully"), nil
}

type replaceMemoryTool struct {
	store    memory.Store
	memoryID string
}

func newReplaceMemoryTool(store memory.Store, memoryID string) *replaceMemoryTool {
	return &replaceMemoryTool{store: store, memoryID: memoryID}
}

func (t *replaceMemoryTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "replace_memory",
		Description: "Replace an existing memory with updated information. Use when a fact has changed or needs correction. First use recall_memories to find the memory_id.",
		Parameters: map[string]any{
			"memory_id": map[string]any{
				"type":        "string",
				"description": "The ID of the memory to replace (from recall_memories results)",
			},
			"fact": map[string]any{
				"type":        "string",
				"description": "The updated fact to store",
			},
			"category": map[string]any{
				"type":        "string",
				"enum":        []string{"preference", "personal", "health", "professional", "other"},
				"description": "Category of the memory",
			},
		},
		Required: []string{"memory_id", "fact"},
	}
}

func (t *replaceMemoryTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input struct {
		MemoryID string `json:"memory_id"`
		Fact     string `json:"fact"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	metadata := map[string]any{}
	if input.Category != "" {
		metadata["category"] = input.Category
	}

	if err := t.store.Update(ctx, input.MemoryID, input.Fact, metadata); err != nil {
		return tool.NewTextErrorResponse("failed to replace memory: " + err.Error()), nil
	}

	return tool.NewTextResponse("Memory replaced successfully"), nil
}

func createMemoryTools(store memory.Store, memoryID string) []tool.BaseTool {
	return []tool.BaseTool{
		newStoreMemoryTool(store, memoryID),
		newRecallMemoriesTool(store, memoryID),
		newReplaceMemoryTool(store, memoryID),
		newDeleteMemoryTool(store, memoryID),
	}
}
