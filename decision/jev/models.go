package jev

import "github.com/joakimcarlsson/ai/decision"

// Model IDs served by this catalog.
const (
	JevLatest string = "jev-latest"
	Jev113    string = "jev-1.13.0"
	Jev114    string = "jev-1.14.0"
)

// Models maps model IDs to their configurations. Input is billed at $0.042
// per 1M tokens; output is free.
var Models = map[string]decision.Model{
	JevLatest: {
		ID:                   JevLatest,
		Name:                 "Jev (latest)",
		Provider:             "typesafe",
		APIModel:             "jev-latest",
		Currency:             "USD",
		CostPer1MInputTokens: 0.042,
	},
	Jev113: {
		ID:                   Jev113,
		Name:                 "Jev 1.13",
		Provider:             "typesafe",
		APIModel:             "jev-1.13.0",
		Currency:             "USD",
		CostPer1MInputTokens: 0.042,
	},
	Jev114: {
		ID:                   Jev114,
		Name:                 "Jev 1.14",
		Provider:             "typesafe",
		APIModel:             "jev-1.14.0",
		Currency:             "USD",
		CostPer1MInputTokens: 0.042,
	},
}
