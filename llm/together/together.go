// Package together provides an OpenAI-compatible LLM client targeting
// Together AI.
//
// This is a thin wrapper over [llm/openai] fixed to Together's
// chat-completions endpoint. Together hosts far more models than the [model]
// package catalogues; callers can pass any hosted model id via
// [llmopenai.WithModel] even without a registered entry.
package together

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Together AI API endpoint.
const DefaultBaseURL = "https://api.together.xyz/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Together AI LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override. Models with CanReason replay each assistant turn's
// reasoning under the field the model returned it in, reasoning or
// reasoning_content, through tool loops and preserved thinking;
// [llmopenai.WithReasoningContentReplay] overrides this.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{
			llmopenai.WithBaseURL(DefaultBaseURL),
			llmopenai.WithReasoningReplayDefault(canReason),
		}, opts...)...)
}

// canReason reports whether the model is a thinking model.
func canReason(m llm.Model) bool { return m.CanReason }

// WithPreservedThinking sets the thinking-history mode for models with
// CanReason. True keeps reasoning from earlier turns in the prompt by sending
// chat_template_kwargs.clear_thinking=false; false sends clear_thinking=true.
// Unset sends nothing and uses the model default.
func WithPreservedThinking(preserved bool) Option {
	return llmopenai.WithReasoningJSONField(
		"chat_template_kwargs.clear_thinking",
		!preserved,
	)
}

// WithThinking toggles reasoning for models that support it. False sends
// reasoning.enabled=false to disable thinking. Unset uses the provider default;
// models without CanReason ignore this option.
func WithThinking(enabled bool) Option {
	return llmopenai.WithReasoningJSONField("reasoning.enabled", enabled)
}
