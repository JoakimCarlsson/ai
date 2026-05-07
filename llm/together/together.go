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
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
