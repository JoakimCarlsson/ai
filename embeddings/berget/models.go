package berget

import (
	"github.com/joakimcarlsson/ai/embeddings"
)

// Berget AI (https://berget.ai) is a Swedish, EU-hosted inference provider with
// an OpenAI-compatible API at https://api.berget.ai/v1.
//
// All prices below are in EUR, not USD: Berget bills in EUR and the Cost*
// fields hold the raw EUR figures from the /v1/models API (fetched 2026-06-30).
// The API does not return context windows, so ContextWindow values come from
// the upstream model cards (131_072 where a model's window is unpublished).
const (
	E5LargeInstruct string = "intfloat/multilingual-e5-large-instruct"
	E5Large         string = "intfloat/multilingual-e5-large"
)

// Models maps Berget embedding model IDs to their configurations.
// CostPer1MTokens is EUR per 1M tokens.
//
// Pricing source: https://api.berget.ai/v1/models.
// Fetched: 2026-07-26.
var Models = map[string]embeddings.EmbeddingModel{
	E5LargeInstruct: {
		ID:              E5LargeInstruct,
		Name:            "Multilingual E5 Large Instruct",
		Provider:        "berget",
		APIModel:        "intfloat/multilingual-e5-large-instruct",
		CostPer1MTokens: 0.03,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
	},
	E5Large: {
		ID:              E5Large,
		Name:            "Multilingual E5 Large",
		Provider:        "berget",
		APIModel:        "intfloat/multilingual-e5-large",
		CostPer1MTokens: 0.03,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
	},
}
