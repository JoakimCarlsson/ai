// Package perplexity provides an OpenAI-compatible LLM client targeting
// Perplexity's Sonar API.
//
// This is a thin wrapper over [llm/openai] fixed to Perplexity's
// chat-completions endpoint. Vendor-unique features (citations metadata,
// search domain filters) are not surfaced; the OpenAI-compatible subset is.
package perplexity

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Perplexity API endpoint.
const DefaultBaseURL = "https://api.perplexity.ai"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Perplexity LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
