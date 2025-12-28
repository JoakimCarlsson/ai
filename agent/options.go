package agent

import (
	"context"

	"github.com/joakimcarlsson/ai/agent/memory"
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
// When set, the agent automatically injects relevant memories into the system prompt.
// Use memory.AutoExtract() to enable automatic fact extraction from conversations.
// Use memory.AutoDedup() to enable LLM-based memory deduplication.
// Use memory.LLM() to set a separate LLM for memory operations.
func WithMemory(mem Memory, opts ...memory.Option) AgentOption {
	return func(a *Agent) {
		a.memory = mem
		cfg := memory.Apply(opts...)
		a.autoExtract = cfg.AutoExtract
		a.autoDedup = cfg.AutoDedup
		if cfg.LLM != nil {
			a.memoryLLM = cfg.LLM
		}
	}
}

// WithUserIDKey sets the context key used to retrieve the user ID for memory operations.
// Default is "user_id". The user ID must be set in the context passed to Chat/ChatStream.
func WithUserIDKey(key string) AgentOption {
	return func(a *Agent) {
		a.userIDKey = key
	}
}

// FileStore creates a file-based session store that persists conversations to disk.
// Sessions are stored as JSON files in the specified directory.
func FileStore(path string) SessionStore {
	store, err := NewFileSessionStore(path)
	if err != nil {
		return nil
	}
	return store
}

// MemoryStore creates an in-memory session store for ephemeral conversations.
// Useful for testing or when persistence is not required.
func MemoryStore() SessionStore {
	return &memorySessionStore{}
}

// WithSession configures the agent with a session for conversation persistence.
// The session is automatically loaded if it exists, or created if it doesn't.
// If not called, the agent operates in stateless mode (no conversation history).
func WithSession(id string, store SessionStore) AgentOption {
	return func(a *Agent) {
		if store == nil {
			return
		}
		ctx := context.Background()
		exists, err := store.Exists(ctx, id)
		if err != nil {
			return
		}
		if exists {
			a.session, _ = store.Load(ctx, id)
		} else {
			a.session, _ = store.Create(ctx, id)
		}
	}
}
