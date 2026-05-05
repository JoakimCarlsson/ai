// Package ollama provides an OpenAI-compatible LLM client targeting a local
// Ollama instance.
//
// This is a thin wrapper over [llm/openai] fixed to Ollama's
// OpenAI-compatible chat-completions endpoint. Ollama does not require an API
// key, so the wrapper defaults to an empty key; callers running behind a
// proxy that does require auth can override with [llmopenai.WithAPIKey].
//
// Ollama hosts whichever models you've pulled locally; the [model] package
// catalogues a representative subset (Llama 3.x, Qwen, DeepSeek-R1, Mistral)
// but callers can pass any pulled model id via [llmopenai.WithModel].
package ollama

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is Ollama's default OpenAI-compatible endpoint.
const DefaultBaseURL = "http://localhost:11434/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs an Ollama LLM client.
//
// The defaults ([llmopenai.WithBaseURL]([DefaultBaseURL]) and
// [llmopenai.WithAPIKey]("")) are prepended; callers can override either by
// passing the same option again in opts.
func NewLLM(opts ...Option) llm.LLM {
	defaults := []Option{
		llmopenai.WithBaseURL(DefaultBaseURL),
		llmopenai.WithAPIKey(""),
	}
	return llmopenai.NewLLM(append(defaults, opts...)...)
}
