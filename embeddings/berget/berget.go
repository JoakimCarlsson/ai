// Package berget provides a Berget AI implementation of the
// [embeddings.Embedding] interface.
//
// Berget's embeddings endpoint is OpenAI-compatible, so this wraps
// [embeddings/openai] pinned to Berget's base URL (https://api.berget.ai/v1).
// See [github.com/joakimcarlsson/ai/model] for the catalog
// (BergetEmbeddingModels) and pricing.
package berget

import (
	"github.com/joakimcarlsson/ai/embeddings"
	embopenai "github.com/joakimcarlsson/ai/embeddings/openai"
)

// DefaultBaseURL is the canonical Berget AI OpenAI-compatible API endpoint.
const DefaultBaseURL = "https://api.berget.ai/v1"

// Option re-exports [embopenai.Option] for caller convenience.
type Option = embopenai.Option

// NewEmbedding constructs a Berget AI embeddings client.
//
// [embopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	return embopenai.NewEmbedding(
		append([]Option{embopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
