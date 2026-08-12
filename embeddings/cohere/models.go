package cohere

import (
	"github.com/joakimcarlsson/ai/embeddings"
)

// Cohere provider identifier and Command model IDs for this registry.
//
// Cohere also serves three Command A variants that are intentionally not
// catalogued here: command-a-reasoning-08-2025, command-a-vision-07-2025 and
// command-a-translate-08-2025. Their per-token rates are sales-gated rather
// than published, so recording a price for them would be a guess.
const (
	EmbedV4           string = "embed-v4.0"
	EmbedMultiV3      string = "embed-multilingual-v3.0"
	EmbedEnV3         string = "embed-english-v3.0"
	EmbedMultiLightV3 string = "embed-multilingual-light-v3.0"
	EmbedEnLightV3    string = "embed-english-light-v3.0"
)

// Models maps Cohere embedding model IDs to their configurations.
//
// Pricing source: https://docs.cohere.com/docs/models.
// Fetched: 2026-07-26.
var Models = map[string]embeddings.EmbeddingModel{
	EmbedV4: {
		ID:              EmbedV4,
		Name:            "Cohere Embed v4.0",
		Provider:        "cohere",
		APIModel:        "embed-v4.0",
		Currency:        "USD",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  128000,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	EmbedMultiV3: {
		ID:              EmbedMultiV3,
		Name:            "Cohere Embed Multilingual v3.0",
		Provider:        "cohere",
		APIModel:        "embed-multilingual-v3.0",
		Currency:        "USD",
		CostPer1MTokens: 0.1,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	EmbedEnV3: {
		ID:              EmbedEnV3,
		Name:            "Cohere Embed English v3.0",
		Provider:        "cohere",
		APIModel:        "embed-english-v3.0",
		Currency:        "USD",
		CostPer1MTokens: 0.1,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	EmbedMultiLightV3: {
		ID:              EmbedMultiLightV3,
		Name:            "Cohere Embed Multilingual Light v3.0",
		Provider:        "cohere",
		APIModel:        "embed-multilingual-light-v3.0",
		Currency:        "USD",
		CostPer1MTokens: 0.1,
		MaxInputTokens:  512,
		EmbeddingDims:   384,
		MaxBatchSize:    96,
	},
	EmbedEnLightV3: {
		ID:              EmbedEnLightV3,
		Name:            "Cohere Embed English Light v3.0",
		Provider:        "cohere",
		APIModel:        "embed-english-light-v3.0",
		Currency:        "USD",
		CostPer1MTokens: 0.1,
		MaxInputTokens:  512,
		EmbeddingDims:   384,
		MaxBatchSize:    96,
	},
}
