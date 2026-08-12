package openai

import (
	"github.com/joakimcarlsson/ai/embeddings"
)

// OpenAI provider plus chat, embedding, and image model IDs for this registry.
const (
	TextEmbedding3Large string = "text-embedding-3-large"
	TextEmbedding3Small string = "text-embedding-3-small"
	AdaEmbedding002     string = "text-embedding-ada-002"
)

// Models maps OpenAI embedding model IDs to their configurations.
//
// Pricing source: https://developers.openai.com/api/docs/pricing (embedding
// rates not published on the page; carried forward).
// Fetched: not re-verified in the 2026-07-26 sweep.
var Models = map[string]embeddings.EmbeddingModel{
	TextEmbedding3Large: {
		ID:              TextEmbedding3Large,
		Name:            "Text Embedding 3 Large",
		Provider:        "openai",
		APIModel:        "text-embedding-3-large",
		CostPer1MTokens: 0.13,
		MaxInputTokens:  8191,
		EmbeddingDims:   3072,
		SupportedDimensions: []int{
			256,
			512,
			1024,
			1536,
			2048,
			3072,
		},
		MaxBatchSize:      2048,
		MaxTokensPerBatch: 1000000,
	},
	TextEmbedding3Small: {
		ID:                  TextEmbedding3Small,
		Name:                "Text Embedding 3 Small",
		Provider:            "openai",
		APIModel:            "text-embedding-3-small",
		CostPer1MTokens:     0.02,
		MaxInputTokens:      8191,
		EmbeddingDims:       1536,
		SupportedDimensions: []int{512, 1536},
		MaxBatchSize:        2048,
		MaxTokensPerBatch:   1000000,
	},
	AdaEmbedding002: {
		ID:                  AdaEmbedding002,
		Name:                "Ada Embedding 002",
		Provider:            "openai",
		APIModel:            "text-embedding-ada-002",
		CostPer1MTokens:     0.1,
		MaxInputTokens:      8191,
		EmbeddingDims:       1536,
		SupportedDimensions: []int{1536},
		MaxBatchSize:        2048,
		MaxTokensPerBatch:   1000000,
	},
}
