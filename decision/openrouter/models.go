package openrouter

import "github.com/joakimcarlsson/ai/decision"

// Model IDs served by this catalog.
const (
	JevLatest string = "~typesafe/jev-latest"
	Jev113    string = "typesafe/jev-1.13"
)

// Models maps OpenRouter's Jev model IDs to their configurations. OpenRouter
// passes TypeSafe's rate through: $0.042 per 1M input tokens, output free.
var Models = map[string]decision.Model{
	JevLatest: {
		ID:                   JevLatest,
		Name:                 "OpenRouter – Jev (latest)",
		Provider:             "openrouter",
		APIModel:             "~typesafe/jev-latest",
		Currency:             "USD",
		CostPer1MInputTokens: 0.042,
	},
	Jev113: {
		ID:                   Jev113,
		Name:                 "OpenRouter – Jev 1.13",
		Provider:             "openrouter",
		APIModel:             "typesafe/jev-1.13",
		Currency:             "USD",
		CostPer1MInputTokens: 0.042,
	},
}
