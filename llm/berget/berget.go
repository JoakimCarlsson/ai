// Package berget provides an OpenAI-compatible LLM client targeting Berget AI.
//
// This wraps [llm/openai] fixed to Berget's chat-completions endpoint
// (https://api.berget.ai/v1). Berget is a Swedish, EU-hosted provider serving
// open-weight models; see [github.com/joakimcarlsson/ai/model] for the catalog
// (BergetModels) and pricing.
package berget

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Berget AI OpenAI-compatible API endpoint.
const DefaultBaseURL = "https://api.berget.ai/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Berget AI LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override.
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{llmopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
