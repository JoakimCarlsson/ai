package voice

import (
	"context"

	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
)

// Option configures a VoiceAgent. Pass options to New.
type Option func(*VoiceAgent)

// WithSystemPrompt sets the system prompt prepended to every LLM call.
func WithSystemPrompt(prompt string) Option {
	return func(v *VoiceAgent) {
		v.systemPrompt = prompt
	}
}

// WithTools registers tools that the LLM may call during a conversation.
// Multiple WithTools options append.
func WithTools(tools ...tool.BaseTool) Option {
	return func(v *VoiceAgent) {
		v.tools = append(v.tools, tools...)
	}
}

// WithMaxToolIterations caps how many tool-call rounds may run inside a single
// assistant turn. Default is 4. Values <= 0 are ignored.
func WithMaxToolIterations(n int) Option {
	return func(v *VoiceAgent) {
		if n > 0 {
			v.maxToolIterations = n
		}
	}
}

// WithFiller enables filler audio that fires when the LLM is slow to produce
// its first content delta. Disabled when Timeout is zero or Message is empty
// (and Source is nil).
func WithFiller(cfg FillerConfig) Option {
	return func(v *VoiceAgent) {
		v.filler = cfg
	}
}

// WithToolSound configures ambient audio that loops while a tool is executing.
// Disabled when cfg.Audio is empty.
func WithToolSound(cfg ToolSoundConfig) Option {
	return func(v *VoiceAgent) {
		v.toolSound = cfg
	}
}

// WithBargeIn sets the barge-in policy. Default is BargeInIgnore.
func WithBargeIn(policy BargeInPolicy) Option {
	return func(v *VoiceAgent) {
		v.bargeIn = policy
	}
}

// WithHooks registers callbacks invoked at synchronous interception points
// during a conversation: lifecycle, user-message commit, LLM call boundary,
// tool-use boundary, tool error. Multiple Hooks structs may be passed; they
// run in registration order and HookModify mutations chain. Pair with
// Conversation.Events for async observation; use hooks when you need to
// mutate or veto.
func WithHooks(hooks ...Hooks) Option {
	return func(v *VoiceAgent) {
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
	return func(v *VoiceAgent) {
		v.contextStrategy = strategy
		v.maxContextTokens = maxContextTokens
	}
}

// WithSession configures a session store and id for this agent. When set,
// the runner loads existing messages from the store at conversation start
// and persists new messages at turn boundaries.
//
// Mirrors agent.WithSession. If store is nil the option is a no-op. If id
// does not exist in the store it is created.
func WithSession(id string, store session.Store) Option {
	return func(v *VoiceAgent) {
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
