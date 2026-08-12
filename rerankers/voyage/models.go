package voyage

import (
	"github.com/joakimcarlsson/ai/rerankers"
)

// Voyage provider identifier plus embedding and reranker model IDs for this registry.
const (
	Rerank25     string = "rerank-2.5"
	Rerank25Lite string = "rerank-2.5-lite"
	Rerank2      string = "rerank-2"
	Rerank2Lite  string = "rerank-2-lite"
	Rerank1      string = "rerank-1"
	RerankLite1  string = "rerank-lite-1"
)

// Models maps Voyage reranker model IDs to their configurations.
//
// Pricing source: https://docs.voyageai.com/docs/pricing.
// Fetched: 2026-07-26.
var Models = map[string]rerankers.RerankerModel{
	Rerank25: {
		ID:              Rerank25,
		Name:            "Rerank 2.5",
		Provider:        "voyage",
		APIModel:        "rerank-2.5",
		CostPer1MTokens: 0.05,
		MaxQueryTokens:  32000,
		MaxTotalTokens:  600000,
	},
	Rerank25Lite: {
		ID:              Rerank25Lite,
		Name:            "Rerank 2.5 Lite",
		Provider:        "voyage",
		APIModel:        "rerank-2.5-lite",
		CostPer1MTokens: 0.02,
		MaxQueryTokens:  32000,
		MaxTotalTokens:  600000,
	},
	Rerank2: {
		ID:              Rerank2,
		Name:            "Rerank 2 [Legacy]",
		Provider:        "voyage",
		APIModel:        "rerank-2",
		CostPer1MTokens: 0.05,
		MaxQueryTokens:  4000,
		MaxTotalTokens:  600000,
	},
	Rerank2Lite: {
		ID:              Rerank2Lite,
		Name:            "Rerank 2 Lite [Legacy]",
		Provider:        "voyage",
		APIModel:        "rerank-2-lite",
		CostPer1MTokens: 0.02,
		MaxQueryTokens:  2000,
		MaxTotalTokens:  600000,
	},
	Rerank1: {
		ID:              Rerank1,
		Name:            "Rerank 1 [Legacy]",
		Provider:        "voyage",
		APIModel:        "rerank-1",
		CostPer1MTokens: 0.05,
		MaxQueryTokens:  2000,
		MaxTotalTokens:  300000,
	},
	RerankLite1: {
		ID:              RerankLite1,
		Name:            "Rerank Lite 1 [Legacy]",
		Provider:        "voyage",
		APIModel:        "rerank-lite-1",
		CostPer1MTokens: 0.02,
		MaxQueryTokens:  1000,
		MaxTotalTokens:  300000,
	},
}
