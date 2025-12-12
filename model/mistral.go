package model

const (
	ProviderMistral ModelProvider = "mistral"

	MistralLarge3  ModelID = "mistral-large-3"
	MistralMedium3 ModelID = "mistral-medium-3"
	Mixtral8x7B    ModelID = "mixtral-8x7b"
	Mistral7B      ModelID = "mistral-7b"
)

var MistralModels = map[ModelID]Model{
	MistralLarge3: {
		ID:                    MistralLarge3,
		Name:                  "Mistral Large 3",
		Provider:              ProviderMistral,
		APIModel:              "mistral-large-3-25-12",
		CostPer1MIn:           0.50,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          1.50,
		ContextWindow:         256_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	MistralMedium3: {
		ID:                    MistralMedium3,
		Name:                  "Mistral Medium 3.1",
		Provider:              ProviderMistral,
		APIModel:              "mistral-medium-3-1-25-08",
		CostPer1MIn:           0.40,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          2.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Mixtral8x7B: {
		ID:                    Mixtral8x7B,
		Name:                  "Mixtral 8x7B Instruct",
		Provider:              ProviderMistral,
		APIModel:              "mixtral-8x7b-instruct",
		CostPer1MIn:           0.24,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.24,
		ContextWindow:         32_768,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Mistral7B: {
		ID:                    Mistral7B,
		Name:                  "Mistral 7B Instruct",
		Provider:              ProviderMistral,
		APIModel:              "mistral-7b-instruct",
		CostPer1MIn:           0.25,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.25,
		ContextWindow:         32_768,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
