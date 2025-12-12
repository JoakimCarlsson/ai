package model

const (
	ProviderMeta ModelProvider = "meta"

	MetaLlama4Maverick ModelID = "llama-4-maverick"
	MetaLlama4Scout    ModelID = "llama-4-scout"
	MetaLlama31405B    ModelID = "llama-3.1-405b"
	MetaLlama3170B     ModelID = "llama-3.1-70b"
	MetaLlama318B      ModelID = "llama-3.1-8b"
)

var MetaModels = map[ModelID]Model{
	MetaLlama4Maverick: {
		ID:                    MetaLlama4Maverick,
		Name:                  "Llama 4 Maverick",
		Provider:              ProviderMeta,
		APIModel:              "meta-llama/llama-4-maverick",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.60,
		ContextWindow:         1_048_576,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	MetaLlama4Scout: {
		ID:                    MetaLlama4Scout,
		Name:                  "Llama 4 Scout",
		Provider:              ProviderMeta,
		APIModel:              "meta-llama/llama-4-scout",
		CostPer1MIn:           0.08,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.30,
		ContextWindow:         327_680,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	MetaLlama31405B: {
		ID:                    MetaLlama31405B,
		Name:                  "Llama 3.1 405B",
		Provider:              ProviderMeta,
		APIModel:              "meta-llama/Meta-Llama-3.1-405B-Instruct",
		CostPer1MIn:           4.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          4.00,
		ContextWindow:         32_768,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	MetaLlama3170B: {
		ID:                    MetaLlama3170B,
		Name:                  "Llama 3.1 70B Instruct",
		Provider:              ProviderMeta,
		APIModel:              "meta-llama/Meta-Llama-3.1-70B-Instruct",
		CostPer1MIn:           0.40,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.40,
		ContextWindow:         131_072,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	MetaLlama318B: {
		ID:                    MetaLlama318B,
		Name:                  "Llama 3.1 8B Instruct",
		Provider:              ProviderMeta,
		APIModel:              "meta-llama/Meta-Llama-3.1-8B-Instruct",
		CostPer1MIn:           0.02,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.03,
		ContextWindow:         131_072,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
