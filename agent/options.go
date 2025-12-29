package agent

import (
	"context"

	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/tokens"
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
// The id parameter identifies the memory owner (e.g., user ID).
// When set, the agent automatically injects relevant memories into the system prompt.
// Use memory.AutoExtract() to enable automatic fact extraction from conversations.
// Use memory.AutoDedup() to enable LLM-based memory deduplication.
// Use memory.LLM() to set a separate LLM for memory operations.
func WithMemory(id string, store memory.Store, opts ...memory.Option) AgentOption {
	return func(a *Agent) {
		a.memoryID = id
		a.memory = store
		cfg := memory.Apply(opts...)
		a.autoExtract = cfg.AutoExtract
		a.autoDedup = cfg.AutoDedup
		if cfg.LLM != nil {
			a.memoryLLM = cfg.LLM
		}
	}
}

// WithSession configures the agent with a session for conversation persistence.
// The session is automatically loaded if it exists, or created if it doesn't.
// If not called, the agent operates in stateless mode (no conversation history).
func WithSession(id string, store session.Store) AgentOption {
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

// WithContextStrategy configures automatic context window management.
// When the conversation exceeds the token limit, the strategy trims messages to fit.
//
// The maxContextTokens parameter sets the maximum tokens allowed for the conversation.
// When the conversation exceeds this limit, the strategy is applied.
//
// Example with truncation:
//
//	agent.WithContextStrategy(truncate.Strategy(), 8000)
//
// Example with sliding window:
//
//	agent.WithContextStrategy(sliding.Strategy(sliding.KeepLast(20)), 8000)
//
// Example with summarization:
//
//	agent.WithContextStrategy(summarize.Strategy(summaryLLM), 8000)
func WithContextStrategy(strategy tokens.Strategy, maxContextTokens int64) AgentOption {
	return func(a *Agent) {
		a.contextStrategy = strategy
		a.maxContextTokens = maxContextTokens
	}
}
