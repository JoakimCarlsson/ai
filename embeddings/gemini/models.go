package gemini

import (
	"github.com/joakimcarlsson/ai/embeddings"
)

// Gemini provider plus Gemini and Imagen model IDs for this registry.
const (
	Embedding2   string = "gemini-embedding-2"
	Embedding001 string = "gemini-embedding-001"
)

// Models maps Gemini embedding model IDs to their configurations.
//
// Pricing source: https://ai.google.dev/gemini-api/docs/pricing.
// Fetched: 2026-07-26.
var Models = map[string]embeddings.EmbeddingModel{
	Embedding2: {
		ID:                  Embedding2,
		Name:                "Gemini Embedding 2",
		Provider:            "gemini",
		APIModel:            "gemini-embedding-2",
		Currency:            "USD",
		CostPer1MTokens:     0.2,
		MaxInputTokens:      8192,
		EmbeddingDims:       768,
		SupportedDimensions: []int{768, 1536, 3072},
		MaxBatchSize:        100,
	},
	Embedding001: {
		ID:                  Embedding001,
		Name:                "Gemini Embedding 001",
		Provider:            "gemini",
		APIModel:            "gemini-embedding-001",
		Currency:            "USD",
		CostPer1MTokens:     0.15,
		MaxInputTokens:      2048,
		EmbeddingDims:       3072,
		SupportedDimensions: []int{768, 1536, 3072},
		MaxBatchSize:        100,
	},
}
