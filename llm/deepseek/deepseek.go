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
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
