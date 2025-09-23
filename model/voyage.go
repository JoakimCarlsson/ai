package model

const (
	ProviderVoyage ModelProvider = "voyage"

	Voyage35       ModelID = "voyage-3.5"
	Voyage35Lite   ModelID = "voyage-3.5-lite"
	Voyage3Large   ModelID = "voyage-3-large"
	VoyageCode3    ModelID = "voyage-code-3"
	VoyageContext3 ModelID = "voyage-context-3"

	Rerank25     ModelID = "rerank-2.5"
	Rerank25Lite ModelID = "rerank-2.5-lite"
	Rerank2      ModelID = "rerank-2"
	Rerank2Lite  ModelID = "rerank-2-lite"
	Rerank1      ModelID = "rerank-1"
	RerankLite1  ModelID = "rerank-lite-1"

	VoyageFinance2 ModelID = "voyage-finance-2"
	VoyageLaw2     ModelID = "voyage-law-2"
	VoyageCode2    ModelID = "voyage-code-2"

	Voyage3              ModelID = "voyage-3"
	Voyage3Lite          ModelID = "voyage-3-lite"
	VoyageMultilingual2  ModelID = "voyage-multilingual-2"
	VoyageLarge2Instruct ModelID = "voyage-large-2-instruct"
	VoyageLarge2         ModelID = "voyage-large-2"
	Voyage2              ModelID = "voyage-2"
	VoyageMulti3         ModelID = "voyage-multimodal-3"

	VoyageLite02Instruct ModelID = "voyage-lite-02-instruct"
	Voyage02             ModelID = "voyage-02"
	Voyage01             ModelID = "voyage-01"
	VoyageLite01         ModelID = "voyage-lite-01"
	VoyageLite01Instruct ModelID = "voyage-lite-01-instruct"
)

var VoyageEmbeddingModels = map[ModelID]EmbeddingModel{
	Voyage3Large: {
		ID:              Voyage3Large,
		Name:            "Voyage 3 Large",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3-large",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024, // Supports 256, 512, 1024 (default), 2048
	},
	Voyage35: {
		ID:              Voyage35,
		Name:            "Voyage 3.5",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3.5",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024, // Supports 256, 512, 1024 (default), 2048
	},
	Voyage35Lite: {
		ID:              Voyage35Lite,
		Name:            "Voyage 3.5 Lite",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-3.5-lite",
		CostPer1MTokens: 0.07,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024, // Supports 256, 512, 1024 (default), 2048
	},
	VoyageCode3: {
		ID:              VoyageCode3,
		Name:            "Voyage Code 3",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-code-3",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024, // Supports 256, 512, 1024 (default), 2048
	},
	VoyageFinance2: {
		ID:              VoyageFinance2,
		Name:            "Voyage Finance 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-finance-2",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024,
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
	VoyageCode2: {
		ID:              VoyageCode2,
		Name:            "Voyage Code 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-code-2",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  16000,
		EmbeddingDims:   1536,
	},
	VoyageContext3: {
		ID:              VoyageContext3,
		Name:            "Voyage Context 3",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-context-3",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024, // Supports 256, 512, 1024 (default), 2048
	},
	// Older models (still supported)
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
	VoyageMultilingual2: {
		ID:              VoyageMultilingual2,
		Name:            "Voyage Multilingual 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-multilingual-2",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  32000,
		EmbeddingDims:   1024,
	},
	VoyageLarge2Instruct: {
		ID:              VoyageLarge2Instruct,
		Name:            "Voyage Large 2 Instruct",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-large-2-instruct",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  16000,
		EmbeddingDims:   1024,
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
	Voyage2: {
		ID:              Voyage2,
		Name:            "Voyage 2",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-2",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  4000,
		EmbeddingDims:   1024,
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

	VoyageLite02Instruct: {
		ID:              VoyageLite02Instruct,
		Name:            "Voyage Lite 02 Instruct [Deprecated]",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-lite-02-instruct",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  4000,
		EmbeddingDims:   1024,
	},
	Voyage02: {
		ID:              Voyage02,
		Name:            "Voyage 02 [Deprecated]",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-02",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  4000,
		EmbeddingDims:   1024,
	},
	Voyage01: {
		ID:              Voyage01,
		Name:            "Voyage 01 [Deprecated]",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-01",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  4000,
		EmbeddingDims:   1024,
	},
	VoyageLite01: {
		ID:              VoyageLite01,
		Name:            "Voyage Lite 01 [Deprecated]",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-lite-01",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  4000,
		EmbeddingDims:   1024,
	},
	VoyageLite01Instruct: {
		ID:              VoyageLite01Instruct,
		Name:            "Voyage Lite 01 Instruct [Deprecated]",
		Provider:        ProviderVoyage,
		APIModel:        "voyage-lite-01-instruct",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  4000,
		EmbeddingDims:   1024,
	},
}

var VoyageRerankerModels = map[ModelID]RerankerModel{
	Rerank25: {
		ID:              Rerank25,
		Name:            "Rerank 2.5",
		Provider:        ProviderVoyage,
		APIModel:        "rerank-2.5",
		CostPer1MTokens: 0.05,
		MaxQueryTokens:  32000,
		MaxTotalTokens:  600000,
	},
	Rerank25Lite: {
		ID:              Rerank25Lite,
		Name:            "Rerank 2.5 Lite",
		Provider:        ProviderVoyage,
		APIModel:        "rerank-2.5-lite",
		CostPer1MTokens: 0.02,
		MaxQueryTokens:  32000,
		MaxTotalTokens:  600000,
	},

	Rerank2: {
		ID:              Rerank2,
		Name:            "Rerank 2 [Legacy]",
		Provider:        ProviderVoyage,
		APIModel:        "rerank-2",
		CostPer1MTokens: 0.05,
		MaxQueryTokens:  4000,
		MaxTotalTokens:  600000,
	},
	Rerank2Lite: {
		ID:              Rerank2Lite,
		Name:            "Rerank 2 Lite [Legacy]",
		Provider:        ProviderVoyage,
		APIModel:        "rerank-2-lite",
		CostPer1MTokens: 0.02,
		MaxQueryTokens:  2000,
		MaxTotalTokens:  600000,
	},
	Rerank1: {
		ID:              Rerank1,
		Name:            "Rerank 1 [Legacy]",
		Provider:        ProviderVoyage,
		APIModel:        "rerank-1",
		CostPer1MTokens: 0.05,
		MaxQueryTokens:  2000,
		MaxTotalTokens:  300000,
	},
	RerankLite1: {
		ID:              RerankLite1,
		Name:            "Rerank Lite 1 [Legacy]",
		Provider:        ProviderVoyage,
		APIModel:        "rerank-lite-1",
		CostPer1MTokens: 0.02,
		MaxQueryTokens:  1000,
		MaxTotalTokens:  300000,
	},
}
