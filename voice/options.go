package voice

import (
	"context"

	"github.com/joakimcarlsson/ai/memory"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
)

// Option configures a Agent. Pass options to New.
type Option func(*Agent)

// WithSystemPrompt sets the system prompt prepended to every LLM call.
func WithSystemPrompt(prompt string) Option {
	return func(v *Agent) {
		v.systemPrompt = prompt
	}
}

// WithTools registers tools that the LLM may call during a conversation.
// Multiple WithTools options append.
func WithTools(tools ...tool.BaseTool) Option {
	return func(v *Agent) {
		v.tools = append(v.tools, tools...)
	}
}

// WithToolsets registers tool.Toolset implementations whose Tools(ctx)
// method is consulted before every LLM call. Toolset tools are appended
// to the static set registered via WithTools; the union is what the LLM
// sees for that turn.
//
// Use toolsets when the available tools depend on per-call context — e.g.,
// MCP servers (tool.MCPToolset), per-user RBAC filtering
// (tool.NewFilterToolset), or feature-flagged sets composed via
// tool.NewCompositeToolset. Static tools that don't change should still
// be passed via WithTools to avoid re-resolving them every turn.
//
// Mirrors agent.WithToolsets.
func WithToolsets(toolsets ...tool.Toolset) Option {
	return func(v *Agent) {
		v.toolsets = append(v.toolsets, toolsets...)
	}
}

// WithMaxToolIterations caps how many tool-call rounds may run inside a single
// assistant turn. Default is 4. Values <= 0 are ignored.
func WithMaxToolIterations(n int) Option {
	return func(v *Agent) {
		if n > 0 {
			v.maxToolIterations = n
		}
	}
}

// WithFiller enables filler audio that fires when the LLM is slow to produce
// its first content delta. Disabled when Timeout is zero or Message is empty
// (and Source is nil).
func WithFiller(cfg FillerConfig) Option {
	return func(v *Agent) {
		v.filler = cfg
	}
}

// WithToolSound configures ambient audio that loops while a tool is executing.
// Disabled when cfg.Audio is empty.
func WithToolSound(cfg ToolSoundConfig) Option {
	return func(v *Agent) {
		v.toolSound = cfg
	}
}

// WithBargeIn sets the barge-in policy. Default is BargeInIgnore.
func WithBargeIn(policy BargeInPolicy) Option {
	return func(v *Agent) {
		v.bargeIn = policy
	}
}

// WithHandoffs registers handoff targets that the LLM may transfer control
// to. For each config a "transfer_to_<Name>" tool is added to v.tools so
// the LLM can invoke the handoff. When the runner detects a handoff tool
// call it swaps the active agent for the rest of the conversation —
// the target's system prompt, tools, LLM, hooks, context strategy, and
// chained handoffs all take over. The target's STT/TTS clients are
// ignored; the audio path stays bound to the original agent.
//
// Mirrors agent.WithHandoffs.
func WithHandoffs(configs ...HandoffConfig) Option {
	return func(v *Agent) {
		v.handoffs = append(v.handoffs, configs...)
		for _, cfg := range configs {
			v.tools = append(v.tools, newHandoffTool(cfg))
		}
	}
}

// WithHooks registers callbacks invoked at synchronous interception points
// during a conversation: lifecycle, user-message commit, LLM call boundary,
// tool-use boundary, tool error. Multiple Hooks structs may be passed; they
// run in registration order and HookModify mutations chain. Pair with
// Conversation.Events for async observation; use hooks when you need to
// mutate or veto.
func WithHooks(hooks ...Hooks) Option {
	return func(v *Agent) {
		v.hooks = append(v.hooks, hooks...)
	}
}

// WithContextStrategy configures automatic context-window management. The
// strategy is invoked before every LLM call inside an assistant turn; when
// the conversation exceeds maxContextTokens it trims, slides, or summarizes
// the message list before it is sent to the model.
//
// If maxContextTokens is <= 0 the option is a no-op until both fields are
// set. The strategy is shared across all conversations on this agent.
//
// Example with sliding:
//
//	voice.WithContextStrategy(sliding.Strategy(sliding.KeepLast(20)), 8000)
//
// Example with summarization:
//
//	voice.WithContextStrategy(summarize.Strategy(summaryLLM), 8000)
//
// Mirrors agent.WithContextStrategy.
func WithContextStrategy(
	strategy tokens.Strategy,
	maxContextTokens int64,
) Option {
	return func(v *Agent) {
		v.contextStrategy = strategy
		v.maxContextTokens = maxContextTokens
	}
}

// WithMemory configures long-term memory recall and (optional) automatic
// fact extraction for this agent. Mirrors agent.WithMemory.
//
// id is the owner identifier (typically a user id) under which memories
// are stored and searched. store is any memory.Store implementation.
//
// Behavior driven by opts:
//   - memory.AutoExtract() — after each successful user turn, the runner
//     fires a background goroutine that calls memory.ExtractFacts on the
//     session messages and persists the results to the store. Requires
//     a session (WithSession) so there's something to extract from.
//   - memory.AutoDedup() — before storing each extracted fact, run
//     memory.Deduplicate against the top-5 nearest existing memories and
//     apply the resulting Add/Update/Delete decisions.
//   - memory.LLM(separate) — use a different LLM for extraction/dedup
//     than the conversation LLM. If unset, uses the agent's main LLM.
//
// When AutoExtract is disabled and a memoryID is set, the agent registers
// the four memory.Tools (store_memory / recall_memories / replace_memory /
// delete_memory) so the LLM can manage memory explicitly via tool calls.
//
// Recall happens before every LLM call: top-5 memories matching the most
// recent user message are prepended as a transient system message; not
// persisted to history or session.
func WithMemory(
	id string,
	store memory.Store,
	opts ...memory.Option,
) Option {
	return func(v *Agent) {
		v.memoryID = id
		v.memory = store
		cfg := memory.Apply(opts...)
		v.autoExtract = cfg.AutoExtract
		v.autoDedup = cfg.AutoDedup
		if cfg.LLM != nil {
			v.memoryLLM = cfg.LLM
		}
	}
}

// WithSession configures a session store and id for this agent. When set,
// the runner loads existing messages from the store at conversation start
// and persists new messages at turn boundaries.
//
// Mirrors agent.WithSession. If store is nil the option is a no-op. If id
// does not exist in the store it is created.
func WithSession(id string, store session.Store) Option {
	return func(v *Agent) {
		if store == nil {
			return
		}
		ctx := context.Background()
		exists, err := store.Exists(ctx, id)
		if err != nil {
			return
		}
		if exists {
			v.session, _ = store.Load(ctx, id)
		} else {
			v.session, _ = store.Create(ctx, id)
		}
	}
}
