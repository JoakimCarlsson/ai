package mistral

import (
	"github.com/joakimcarlsson/ai/embeddings"
)

// Mistral provider identifier and model IDs for this registry.
const (
	Embed          string = "mistral-embed"
	CodestralEmbed string = "codestral-embed"
)

// Models maps Mistral embedding model IDs to their configurations.
//
// Pricing source: https://mistral.ai/pricing/api.
// Fetched: 2026-07-26.
var Models = map[string]embeddings.EmbeddingModel{
	Embed: {
		ID:              Embed,
		Name:            "Mistral Embed",
		Provider:        "mistral",
		APIModel:        "mistral-embed",
		Currency:        "USD",
		CostPer1MTokens: 0.1,
		MaxInputTokens:  8192,
		EmbeddingDims:   1024,
		MaxBatchSize:    512,
	},
	CodestralEmbed: {
		ID:              CodestralEmbed,
		Name:            "Codestral Embed",
		Provider:        "mistral",
		APIModel:        "codestral-embed",
		Currency:        "USD",
		CostPer1MTokens: 0.15,
		MaxInputTokens:  32768,
		EmbeddingDims:   1536,
		SupportedDimensions: []int{
			1536,
			1024,
			768,
			512,
			256,
		},
		MaxBatchSize:        512,
		SupportsOutputDtype: true,
	},
}
