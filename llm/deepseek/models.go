package deepseek

import (
	"github.com/joakimcarlsson/ai/llm"
)

// DeepSeek provider identifier and model IDs for this registry.
const (
	V4Flash     string = "deepseek-v4-flash"
	V4Pro       string = "deepseek-v4-pro"
	V32         string = "deepseek-v3.2"
	V32Thinking string = "deepseek-v3.2-thinking"
	R1          string = "deepseek-r1"
	R1Distill   string = "deepseek-r1-distill-llama-70b"
)

// Models maps DeepSeek model IDs to their configurations.
//
// Pricing source: https://api-docs.deepseek.com/quick_start/pricing.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	V4Flash: {
		ID:                V4Flash,
		Name:              "DeepSeek V4 Flash",
		Provider:          "deepseek",
		APIModel:          "deepseek-v4-flash",
		Currency:          "USD",
		CostPer1MIn:       0.14,
		CostPer1MOut:      0.28,
		CostPer1MInCached: 0.0028,
		ContextWindow:     1000000,
		DefaultMaxTokens:  64000,
		CanReason:         true,
	},
	V4Pro: {
		ID:                V4Pro,
		Name:              "DeepSeek V4 Pro",
		Provider:          "deepseek",
		APIModel:          "deepseek-v4-pro",
		Currency:          "USD",
		CostPer1MIn:       0.435,
		CostPer1MOut:      0.87,
		CostPer1MInCached: 0.003625,
		ContextWindow:     1000000,
		DefaultMaxTokens:  64000,
		CanReason:         true,
	},
	V32: {
		ID:                V32,
		Name:              "DeepSeek V3.2",
		Provider:          "deepseek",
		APIModel:          "deepseek-v3.2",
		Currency:          "USD",
		CostPer1MIn:       0.28,
		CostPer1MOut:      0.42,
		CostPer1MInCached: 0.028,
		ContextWindow:     128000,
		DefaultMaxTokens:  8000,
	},
	V32Thinking: {
		ID:                V32Thinking,
		Name:              "DeepSeek V3.2 Thinking",
		Provider:          "deepseek",
		APIModel:          "deepseek-v3.2-thinking",
		Currency:          "USD",
		CostPer1MIn:       0.28,
		CostPer1MOut:      0.42,
		CostPer1MInCached: 0.028,
		ContextWindow:     128000,
		DefaultMaxTokens:  64000,
		CanReason:         true,
	},
	R1: {
		ID:               R1,
		Name:             "DeepSeek R1",
		Provider:         "deepseek",
		APIModel:         "deepseek-r1",
		Currency:         "USD",
		CostPer1MIn:      0.14,
		CostPer1MOut:     0.14,
		ContextWindow:    128000,
		DefaultMaxTokens: 50000,
		CanReason:        true,
	},
	R1Distill: {
		ID:               R1Distill,
		Name:             "DeepSeek R1 Distill Llama 70B",
		Provider:         "deepseek",
		APIModel:         "deepseek-r1-distill-llama-70b",
		Currency:         "USD",
		CostPer1MIn:      0.14,
		CostPer1MOut:     0.28,
		ContextWindow:    128000,
		DefaultMaxTokens: 50000,
		CanReason:        true,
	},
}
