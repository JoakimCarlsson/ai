package cohere

import (
	"github.com/joakimcarlsson/ai/rerankers"
)

// Cohere provider identifier and Command model IDs for this registry.
//
// Cohere also serves three Command A variants that are intentionally not
// catalogued here: command-a-reasoning-08-2025, command-a-vision-07-2025 and
// command-a-translate-08-2025. Their per-token rates are sales-gated rather
// than published, so recording a price for them would be a guess.
const (
	Rerank4Pro    string = "rerank-v4.0-pro"
	Rerank4Fast   string = "rerank-v4.0-fast"
	Rerank35      string = "rerank-v3.5"
	RerankMultiV3 string = "rerank-multilingual-v3.0"
	RerankEnV3    string = "rerank-english-v3.0"
)

// Models maps Cohere reranker model IDs to their configurations.
//
// Pricing source: https://docs.cohere.com/docs/models.
// Fetched: 2026-07-26.
var Models = map[string]rerankers.RerankerModel{
	Rerank4Pro: {
		ID:              Rerank4Pro,
		Name:            "Cohere Rerank v4.0 Pro",
		Provider:        "cohere",
		APIModel:        "rerank-v4.0-pro",
		CostPer1MTokens: 2.5,
		MaxQueryTokens:  4096,
		MaxTotalTokens:  32768,
	},
	Rerank4Fast: {
		ID:              Rerank4Fast,
		Name:            "Cohere Rerank v4.0 Fast",
		Provider:        "cohere",
		APIModel:        "rerank-v4.0-fast",
		CostPer1MTokens: 2,
		MaxQueryTokens:  4096,
		MaxTotalTokens:  32768,
	},
	Rerank35: {
		ID:              Rerank35,
		Name:            "Cohere Rerank v3.5",
		Provider:        "cohere",
		APIModel:        "rerank-v3.5",
		CostPer1MTokens: 2,
		MaxQueryTokens:  2048,
		MaxTotalTokens:  4096,
	},
	RerankMultiV3: {
		ID:              RerankMultiV3,
		Name:            "Cohere Rerank Multilingual v3.0",
		Provider:        "cohere",
		APIModel:        "rerank-multilingual-v3.0",
		CostPer1MTokens: 2,
		MaxQueryTokens:  2048,
		MaxTotalTokens:  4096,
	},
	RerankEnV3: {
		ID:              RerankEnV3,
		Name:            "Cohere Rerank English v3.0",
		Provider:        "cohere",
		APIModel:        "rerank-english-v3.0",
		CostPer1MTokens: 2,
		MaxQueryTokens:  2048,
		MaxTotalTokens:  4096,
	},
}
