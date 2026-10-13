// Package deepseek provides an OpenAI-compatible LLM client targeting DeepSeek.
//
// This is a thin wrapper over [llm/openai] fixed to DeepSeek's chat-completions
// endpoint. DeepSeek's reasoning_content streaming field is surfaced through
// the openai package's existing thinking-delta event handling.
//
// DeepSeek reports cache usage as top-level usage fields rather than in
// prompt_tokens_details; the openai client reads prompt_cache_hit_tokens into
// [llm.TokenUsage].CacheReadTokens and completion_tokens_details.reasoning_tokens
// into [llm.TokenUsage].ReasoningTokens, so no DeepSeek-specific wiring is needed.
package deepseek

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical DeepSeek API endpoint.
const DefaultBaseURL = "https://api.deepseek.com/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a DeepSeek LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override. Models with CanReason replay each assistant turn's
// reasoning_content unchanged, as thinking-mode conversations with tools
// require; [llmopenai.WithReasoningContentReplay] overrides this.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{
			llmopenai.WithBaseURL(DefaultBaseURL),
			llmopenai.WithReasoningReplayDefault(canReason),
		}, opts...)...)
}

// canReason reports whether the model is a thinking model.
func canReason(m llm.Model) bool { return m.CanReason }

// ReasoningEffort controls DeepSeek thinking depth.
type ReasoningEffort string

// ReasoningEffort values supported by DeepSeek reasoning models.
const (
	ReasoningEffortNone ReasoningEffort = "none"
	ReasoningEffortLow  ReasoningEffort = "low"
	ReasoningEffortHigh ReasoningEffort = "high"
	ReasoningEffortMax  ReasoningEffort = "max"
)

// WithThinking toggles thinking on models that support it. False disables
// thinking with thinking.type=disabled; unset preserves the provider default.
// Models without CanReason ignore this option.
func WithThinking(enabled bool) Option {
	mode := "disabled"
	if enabled {
		mode = "enabled"
	}
	return llmopenai.WithReasoningJSONField("thinking.type", mode)
}

// WithReasoningEffort sets DeepSeek thinking effort. None disables thinking;
// other values enable it and set reasoning_effort. Unset preserves the provider
// default. Models without CanReason ignore this option.
func WithReasoningEffort(effort ReasoningEffort) Option {
	return func(o *llmopenai.Options) {
		WithThinking(effort != ReasoningEffortNone)(o)
		llmopenai.WithReasoningJSONField("reasoning_effort", effort)(o)
	}
}
