// Package fireworks provides an OpenAI-compatible LLM client targeting
// Fireworks AI.
//
// This is a thin wrapper over [llm/openai] fixed to the Fireworks
// chat-completions endpoint. Fireworks hosts far more models than the [model]
// package catalogues; callers can pass any "accounts/.../models/..." path via
// [llmopenai.WithModel] even without a registered entry.
package fireworks

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Fireworks API endpoint.
const DefaultBaseURL = "https://api.fireworks.ai/inference/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Fireworks LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override. Models with CanReason replay each assistant turn's
// reasoning_content unchanged, which interleaved thinking requires after tool
// results; [llmopenai.WithReasoningContentReplay] overrides this. Encrypted
// reasoning is not sent: Fireworks does not support reasoning.encrypted_content.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewChatCompletionsLLM(
		append([]Option{
			llmopenai.WithBaseURL(DefaultBaseURL),
			llmopenai.WithReasoningReplayDefault(canReason),
		}, opts...)...)
}

// canReason reports whether the model is a thinking model.
func canReason(m llm.Model) bool { return m.CanReason }

// ReasoningHistory selects how earlier reasoning is kept across user turns.
type ReasoningHistory string

// ReasoningHistory values.
const (
	// ReasoningHistoryPreserved keeps reasoning from all prior turns.
	ReasoningHistoryPreserved ReasoningHistory = "preserved"
)

// WithReasoningHistory sets reasoning_history for models with CanReason.
// Unset sends nothing.
func WithReasoningHistory(mode ReasoningHistory) Option {
	return llmopenai.WithReasoningJSONField("reasoning_history", mode)
}
