package berget

import (
	"github.com/joakimcarlsson/ai/rerankers"
)

// Berget AI (https://berget.ai) is a Swedish, EU-hosted inference provider with
// an OpenAI-compatible API at https://api.berget.ai/v1.
//
// All prices below are in EUR, not USD: Berget bills in EUR and the Cost*
// fields hold the raw EUR figures from the /v1/models API (fetched 2026-06-30).
// The API does not return context windows, so ContextWindow values come from
// the upstream model cards (131_072 where a model's window is unpublished).
const (
	BGERerankerV2M3 string = "BAAI/bge-reranker-v2-m3"
)

// Models maps Berget reranker model IDs to their configurations.
// CostPer1MTokens is EUR per 1M tokens.
//
// Pricing source: https://api.berget.ai/v1/models.
// Fetched: 2026-07-26.
var Models = map[string]rerankers.RerankerModel{
	BGERerankerV2M3: {
		ID:              BGERerankerV2M3,
		Name:            "BGE Reranker v2 m3",
		Provider:        "berget",
		APIModel:        "BAAI/bge-reranker-v2-m3",
		CostPer1MTokens: 0.1,
		MaxQueryTokens:  512,
		MaxTotalTokens:  8192,
	},
}
