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
func WithMemory(
	id string,
	store memory.Store,
	opts ...memory.Option,
) AgentOption {
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
func WithContextStrategy(
	strategy tokens.Strategy,
	maxContextTokens int64,
) AgentOption {
	return func(a *Agent) {
		a.contextStrategy = strategy
		a.maxContextTokens = maxContextTokens
	}
}

// WithSequentialToolExecution disables parallel tool execution.
// By default, tools are executed in parallel for better performance.
// Use this option when tools have dependencies on each other or when
// you need deterministic execution order.
func WithSequentialToolExecution() AgentOption {
	return func(a *Agent) {
		a.parallelTools = false
	}
}

// WithMaxParallelTools sets the maximum number of tools that can execute concurrently.
// Default is 0 (unlimited). Set to a positive number to limit concurrency.
// This is useful when tools consume significant resources (e.g., API rate limits).
func WithMaxParallelTools(max int) AgentOption {
	return func(a *Agent) {
		if max > 0 {
			a.maxParallelTools = max
		}
	}
}

// WithState sets the state map for template variable substitution in the system prompt.
// Use Go text/template syntax like {{.name}} in the system prompt, and they will be
// replaced with values from this state map. Supports conditionals, loops, and complex data.
func WithState(state map[string]any) AgentOption {
	return func(a *Agent) {
		a.state = state
	}
}

// InstructionProvider is a function that generates the system prompt dynamically.
type InstructionProvider func(ctx context.Context, state map[string]any) (string, error)

// WithInstructionProvider sets a dynamic instruction provider that generates the system
// prompt at runtime. When set, this takes precedence over the static system prompt.
// The provider receives the current context and state map.
func WithInstructionProvider(provider InstructionProvider) AgentOption {
	return func(a *Agent) {
		a.instructionProvider = provider
	}
}

// WithSubAgents registers child agents that the parent agent can invoke as tools.
// Each sub-agent appears as a callable tool to the LLM. When invoked, the sub-agent
// runs its own Chat() loop with a fresh context window and returns the result.
//
// Sub-agents support background execution: the LLM can pass background: true to launch
// a sub-agent asynchronously and collect results later via the auto-registered
// get_task_result and stop_task tools.
//
// Sub-agents do NOT inherit the parent's conversation history, tools, or system prompt.
// They operate as independent agents configured at creation time.
//
// If the parent has an observer set, it is automatically propagated to sub-agents
// that do not already have their own observer.
func WithSubAgents(configs ...SubAgentConfig) AgentOption {
	return func(a *Agent) {
		for _, cfg := range configs {
			if a.observer != nil && cfg.Agent.observer == nil {
				cfg.Agent.observer = a.observer
			}
			a.tools = append(a.tools, newSubAgentTool(cfg))
		}
		if a.taskManager == nil {
			a.taskManager = newTaskManager()
		}
		if a.observer != nil {
			a.taskManager.observer = a.observer
		}
	}
}

// WithObserver attaches a runtime observer that receives telemetry events
// for task lifecycle, turn boundaries, and tool execution.
func WithObserver(obs Observer) AgentOption {
	return func(a *Agent) {
		a.observer = obs
		if a.taskManager != nil {
			a.taskManager.observer = obs
		}
	}
}

// WithHandoffs registers peer agents that this agent can transfer control to.
// When the LLM calls a transfer tool, the conversation continues with the new agent.
// The new agent inherits the full message history but uses its own system prompt and tools.
//
// Handoff agents can themselves have handoffs, enabling chains like A -> B -> C.
func WithHandoffs(configs ...HandoffConfig) AgentOption {
	return func(a *Agent) {
		a.handoffs = append(a.handoffs, configs...)
		for _, cfg := range configs {
			a.tools = append(a.tools, newHandoffTool(cfg))
		}
	}
}

// WithFanOut registers a fan-out tool that spawns multiple sub-agents in parallel.
// The LLM calls this tool with a list of tasks, and each task is dispatched to a
// separate execution of the template agent. Results are aggregated into a single response.
//
// Note: The template agent should not use sessions, as concurrent Chat() calls would race.
func WithFanOut(configs ...FanOutConfig) AgentOption {
	return func(a *Agent) {
		for _, cfg := range configs {
			a.tools = append(a.tools, newFanOutTool(cfg))
		}
	}
}
