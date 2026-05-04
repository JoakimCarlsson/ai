// Package groq provides an OpenAI-compatible LLM client targeting Groq.
//
// This is a thin wrapper over [llm/openai] fixed to Groq's chat-completions
// endpoint. Vendor-specific extensions (Groq's compound systems, etc.) are not
// exposed here.
package groq

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Groq OpenAI-compatible API endpoint.
const DefaultBaseURL = "https://api.groq.com/openai/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Groq LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
