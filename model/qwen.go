package model

const (
	ProviderQwen ModelProvider = "qwen"

	Qwen3Max       ModelID = "qwen-3-max"
	Qwen3Coder480B ModelID = "qwen-3-coder-480b"
	Qwen3CoderPlus ModelID = "qwen-3-coder-plus"
)

var QwenModels = map[ModelID]Model{
	Qwen3Max: {
		ID:                    Qwen3Max,
		Name:                  "Qwen 3 Max",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3-max",
		CostPer1MIn:           1.20,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          6.00,
		ContextWindow:         256_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Qwen3Coder480B: {
		ID:                    Qwen3Coder480B,
		Name:                  "Qwen 3 Coder 480B",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3-coder-480b",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          2.00,
		ContextWindow:         256_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Qwen3CoderPlus: {
		ID:                    Qwen3CoderPlus,
		Name:                  "Qwen 3 Coder Plus",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3-coder-plus",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          2.00,
		ContextWindow:         1_000_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
