package model

const (
	ProviderDeepSeek ModelProvider = "deepseek"

	DeepSeekV32         ModelID = "deepseek-v3.2"
	DeepSeekV32Thinking ModelID = "deepseek-v3.2-thinking"
	DeepSeekR1          ModelID = "deepseek-r1"
	DeepSeekR1Distill   ModelID = "deepseek-r1-distill-llama-70b"
)

var DeepSeekModels = map[ModelID]Model{
	DeepSeekV32: {
		ID:                    DeepSeekV32,
		Name:                  "DeepSeek V3.2",
		Provider:              ProviderDeepSeek,
		APIModel:              "deepseek-v3.2",
		CostPer1MIn:           0.28,
		CostPer1MInCached:     0.028,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.42,
		ContextWindow:         128_000,
		DefaultMaxTokens:      8000,
		CanReason:             false,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	DeepSeekV32Thinking: {
		ID:                    DeepSeekV32Thinking,
		Name:                  "DeepSeek V3.2 Thinking",
		Provider:              ProviderDeepSeek,
		APIModel:              "deepseek-v3.2-thinking",
		CostPer1MIn:           0.28,
		CostPer1MInCached:     0.028,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.42,
		ContextWindow:         128_000,
		DefaultMaxTokens:      64000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	DeepSeekR1: {
		ID:                    DeepSeekR1,
		Name:                  "DeepSeek R1",
		Provider:              ProviderDeepSeek,
		APIModel:              "deepseek-r1",
		CostPer1MIn:           0.14,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.14,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	DeepSeekR1Distill: {
		ID:                    DeepSeekR1Distill,
		Name:                  "DeepSeek R1 Distill Llama 70B",
		Provider:              ProviderDeepSeek,
		APIModel:              "deepseek-r1-distill-llama-70b",
		CostPer1MIn:           0.14,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.28,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
