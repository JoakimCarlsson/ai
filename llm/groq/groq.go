// Package groq provides LLM clients targeting Groq.
//
// [NewLLM] is a thin wrapper over [llm/openai] for users who just want
// Groq's fast OpenAI-compatible chat-completions endpoint. [NewCompoundLLM]
// is a standalone client purpose-built for Groq's compound models, with typed
// support for the server-side built-in tools (browser_search, code_execution,
// visit_website) and structured executed_tools metadata.
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
	return llmopenai.NewLLM(
		append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
