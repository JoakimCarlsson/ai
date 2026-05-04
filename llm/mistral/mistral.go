// Package mistral provides an OpenAI-compatible LLM client targeting Mistral AI.
//
// This is a thin wrapper over [llm/openai] fixed to Mistral's chat-completions
// endpoint. Mistral-specific extensions (FIM, document understanding) live in
// other modules ([fim/mistral], etc.).
package mistral

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Mistral API endpoint.
const DefaultBaseURL = "https://api.mistral.ai/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Mistral LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
