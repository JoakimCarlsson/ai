// Package openrouter provides an OpenAI-compatible LLM client targeting OpenRouter.
//
// OpenRouter is a routing layer over many providers; it speaks the OpenAI
// chat-completions wire format, so this package is a thin wrapper over
// [llm/openai] that fixes the base URL to https://openrouter.ai/api/v1.
//
// OpenRouter exposes far more models than the [model] package catalogues; pass
// any OpenRouter model id via [llmopenai.WithModel] even without a registered
// entry in [model].
package openrouter

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical OpenRouter API endpoint.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs an OpenRouter LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override (e.g. to point at a regional endpoint).
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
