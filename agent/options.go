package agent

import (
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

// AgentOption is a functional option for configuring an Agent.
type AgentOption func(*Agent)

// WithSystemPrompt sets the system prompt that defines the agent's behavior and personality.
func WithSystemPrompt(prompt string) AgentOption {
	return func(a *Agent) {
		a.systemPrompt = prompt
	}
}

// WithTools adds tools that the agent can use during conversations.
// Tools are executed automatically when the LLM requests them (unless WithAutoExecute is false).
func WithTools(tools ...tool.BaseTool) AgentOption {
	return func(a *Agent) {
		a.tools = append(a.tools, tools...)
	}
}

// WithMaxIterations sets the maximum number of tool execution iterations per chat.
// Default is 10. Prevents infinite loops when tools keep triggering more tool calls.
func WithMaxIterations(max int) AgentOption {
	return func(a *Agent) {
		a.maxIterations = max
	}
}

// WithAutoExecute controls whether tools are automatically executed when requested by the LLM.
// Default is true. Set to false for manual tool execution control.
func WithAutoExecute(auto bool) AgentOption {
	return func(a *Agent) {
		a.autoExecute = auto
	}
}

// WithMemory sets the memory store for cross-conversation fact storage.
// When set, the agent gains access to store_memory, recall_memories, replace_memory,
// and delete_memory tools for managing user facts across sessions.
func WithMemory(memory Memory) AgentOption {
	return func(a *Agent) {
		a.memory = memory
	}
}

// WithUserIDKey sets the context key used to retrieve the user ID for memory operations.
// Default is "user_id". The user ID must be set in the context passed to Chat/ChatStream.
func WithUserIDKey(key string) AgentOption {
	return func(a *Agent) {
		a.userIDKey = key
	}
}

// WithAutoExtract enables automatic fact extraction from conversations.
// When enabled, the agent uses an LLM to extract relevant facts from each conversation
// and stores them in the memory store. Requires WithMemory to be set.
func WithAutoExtract(enabled bool) AgentOption {
	return func(a *Agent) {
		a.autoExtract = enabled
	}
}

// WithAutoDedup enables LLM-based memory deduplication on store.
// When enabled, before storing a new memory, the agent searches for similar existing
// memories and asks an LLM to decide whether to ADD, UPDATE, DELETE, or skip.
// Requires WithMemory to be set.
func WithAutoDedup(enabled bool) AgentOption {
	return func(a *Agent) {
		a.autoDedup = enabled
	}
}

// WithMemoryLLM sets a separate LLM for memory operations (extraction and deduplication).
// Useful for using a cheaper or faster model for background memory tasks while keeping
// the main conversation on a more capable model.
func WithMemoryLLM(memoryLLM llm.LLM) AgentOption {
	return func(a *Agent) {
		a.memoryLLM = memoryLLM
	}
}
