// Package cerebras provides an OpenAI-compatible LLM client targeting
// Cerebras Inference.
//
// This is a thin wrapper over [llm/openai] fixed to Cerebras' chat-completions
// endpoint. Cerebras' published model catalogue is small and changes often;
// callers can pass any model id via [llmopenai.WithModel] even without a
// registered entry in [model].
package cerebras

import (
	"strings"

	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Cerebras API endpoint.
const DefaultBaseURL = "https://api.cerebras.ai/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Cerebras LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override. Qwen models replay each assistant turn's reasoning field,
// which their historical-thinking behavior uses; other model families receive
// no reasoning history. [llmopenai.WithReasoningContentReplay] overrides this.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewChatCompletionsLLM(
		append([]Option{
			llmopenai.WithBaseURL(DefaultBaseURL),
			llmopenai.WithReasoningReplayDefault(isQwen),
		}, opts...)...)
}

// isQwen reports whether the model belongs to the Qwen family.
func isQwen(m llm.Model) bool {
	return strings.HasPrefix(strings.ToLower(m.APIModel), "qwen")
}

// WithClearThinking sets clear_thinking for Qwen models. True removes
// historical assistant reasoning before the model is prompted; false preserves
// it, which is the provider default. Other model families ignore it. Unset
// sends nothing.
func WithClearThinking(clearHistory bool) Option {
	return llmopenai.WithModelRequestJSONField(isQwen, "clear_thinking", clearHistory)
}
