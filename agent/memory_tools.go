package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joakimcarlsson/ai/tool"
)

type storeMemoryTool struct {
	memory    Memory
	userIDKey string
}

func newStoreMemoryTool(memory Memory, userIDKey string) *storeMemoryTool {
	return &storeMemoryTool{memory: memory, userIDKey: userIDKey}
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

	userID, ok := ctx.Value(t.userIDKey).(string)
	if !ok || userID == "" {
		return tool.NewTextErrorResponse("user_id not found in context"), nil
	}

	metadata := map[string]any{}
	if input.Category != "" {
		metadata["category"] = input.Category
	}

	if err := t.memory.Store(ctx, userID, input.Fact, metadata); err != nil {
		return tool.NewTextErrorResponse("failed to store memory: " + err.Error()), nil
	}

	return tool.NewTextResponse("Memory stored successfully"), nil
}

type recallMemoriesTool struct {
	memory    Memory
	userIDKey string
}

func newRecallMemoriesTool(memory Memory, userIDKey string) *recallMemoriesTool {
	return &recallMemoriesTool{memory: memory, userIDKey: userIDKey}
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

	userID, ok := ctx.Value(t.userIDKey).(string)
	if !ok || userID == "" {
		return tool.NewTextErrorResponse("user_id not found in context"), nil
	}

	memories, err := t.memory.Search(ctx, userID, input.Query, 5)
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
	memory    Memory
	userIDKey string
}

func newDeleteMemoryTool(memory Memory, userIDKey string) *deleteMemoryTool {
	return &deleteMemoryTool{memory: memory, userIDKey: userIDKey}
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

	if err := t.memory.Delete(ctx, input.MemoryID); err != nil {
		return tool.NewTextErrorResponse("failed to delete memory: " + err.Error()), nil
	}

	return tool.NewTextResponse("Memory deleted successfully"), nil
}

func createMemoryTools(memory Memory, userIDKey string) []tool.BaseTool {
	return []tool.BaseTool{
		newStoreMemoryTool(memory, userIDKey),
		newRecallMemoriesTool(memory, userIDKey),
		newDeleteMemoryTool(memory, userIDKey),
	}
}
