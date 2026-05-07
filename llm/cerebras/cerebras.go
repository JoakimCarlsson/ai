// Package cerebras provides an OpenAI-compatible LLM client targeting
// Cerebras Inference.
//
// This is a thin wrapper over [llm/openai] fixed to Cerebras' chat-completions
// endpoint. Cerebras' published model catalogue is small and changes often;
// callers can pass any model id via [llmopenai.WithModel] even without a
// registered entry in [model].
package cerebras

import (
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
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
