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
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
