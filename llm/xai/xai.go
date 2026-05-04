// Package xai provides an OpenAI-compatible LLM client targeting xAI's Grok API.
//
// This is a thin wrapper over [llm/openai]. xAI exposes an OpenAI-compatible
// chat-completions endpoint, so the entire request/response pipeline lives in
// llm/openai; this package only fixes the base URL.
//
// Vendor-unique non-OpenAI features (e.g. xAI Live Search) are not exposed here.
package xai

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical xAI API endpoint.
const DefaultBaseURL = "https://api.x.ai/v1"

// Option re-exports [llmopenai.Option] so callers configure the wrapper using
// the same With* helpers they would use against [llm/openai] directly.
type Option = llmopenai.Option

// NewLLM constructs an xAI LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to point at a different endpoint (e.g. a corporate proxy).
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
