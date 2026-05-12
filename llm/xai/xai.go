// Package xai provides LLM clients targeting xAI's Grok API.
//
// [NewLLM] is a thin wrapper over [llm/openai] for users who just want
// xAI's OpenAI-compatible chat-completions endpoint. [NewResponsesLLM] is a
// standalone client targeting xAI's Responses API, with typed support for
// the server-side built-in tools (web_search, x_search, code_execution) and
// structured citation metadata.
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
	return llmopenai.NewLLM(
		append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
