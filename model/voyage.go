package model

const (
	ProviderVoyage ModelProvider = "voyage"

	Voyage35     ModelID = "voyage-3.5"
	Voyage35Lite ModelID = "voyage-3.5-lite"
	Voyage3Large ModelID = "voyage-3-large"
	VoyageCode3  ModelID = "voyage-code-3"

	VoyageCode2    ModelID = "voyage-code-2"
	VoyageLarge2   ModelID = "voyage-large-2"
	VoyageLaw2     ModelID = "voyage-law-2"
	VoyageFinance2 ModelID = "voyage-finance-2"
	Voyage3        ModelID = "voyage-3"
	Voyage3Lite    ModelID = "voyage-3-lite"
	VoyageMulti3   ModelID = "voyage-multimodal-3"
)

var VoyageEmbeddingModels = map[ModelID]EmbeddingModel{
	Voyage35: {
		ID:              Voyage35,
		Name:            "Voyage 3.5",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3.5",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  320000,
		EmbeddingDims:   1024,
	},
	Voyage35Lite: {
		ID:              Voyage35Lite,
		Name:            "Voyage 3.5 Lite",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3.5-lite",
		CostPer1MTokens: 0.07,
		MaxInputTokens:  1000000,
		EmbeddingDims:   1024,
	},
	Voyage3Large: {
		ID:              Voyage3Large,
		Name:            "Voyage 3 Large",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3-large",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  120000,
		EmbeddingDims:   1024,
	},
	VoyageCode3: {
		ID:              VoyageCode3,
		Name:            "Voyage Code 3",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-code-3",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  120000,
		EmbeddingDims:   1024,
	},

	VoyageCode2: {
		ID:              VoyageCode2,
		Name:            "Voyage Code 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-code-2",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  16000,
		EmbeddingDims:   1536,
	},
	VoyageLarge2: {
		ID:              VoyageLarge2,
		Name:            "Voyage Large 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-large-2",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  16000,
		EmbeddingDims:   1536,
	},
	VoyageLaw2: {
		ID:              VoyageLaw2,
		Name:            "Voyage Law 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-law-2",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  16000,
		EmbeddingDims:   1024,
	},
	VoyageFinance2: {
		ID:              VoyageFinance2,
		Name:            "Voyage Finance 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-finance-2",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  16000,
		EmbeddingDims:   1024,
	},
	Voyage3: {
		ID:              Voyage3,
		Name:            "Voyage 3",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024,
	},
	Voyage3Lite: {
		ID:              Voyage3Lite,
		Name:            "Voyage 3 Lite",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3-lite",
		CostPer1MTokens: 0.07,
		MaxInputTokens:  32000,
		EmbeddingDims:   512,
	},
	VoyageMulti3: {
		ID:              VoyageMulti3,
		Name:            "Voyage Multimodal 3",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-multimodal-3",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024,
	},
}
