package model

// Cohere provider identifier and Command model IDs for this registry.
const (
	ProviderCohere Provider = "cohere"

	CommandAPlus ID = "command-a-plus-05-2026"
	CommandA     ID = "command-a-03-2025"
	CommandR7B   ID = "command-r7b-12-2024"
	CommandRPlus ID = "command-r-plus"
	CommandR     ID = "command-r"

	CohereEmbedV4           ID = "embed-v4.0"
	CohereEmbedMultiV3      ID = "embed-multilingual-v3.0"
	CohereEmbedEnV3         ID = "embed-english-v3.0"
	CohereEmbedMultiLightV3 ID = "embed-multilingual-light-v3.0"
	CohereEmbedEnLightV3    ID = "embed-english-light-v3.0"

	CohereRerank4Pro    ID = "rerank-v4.0-pro"
	CohereRerank4Fast   ID = "rerank-v4.0-fast"
	CohereRerank35      ID = "rerank-v3.5"
	CohereRerankMultiV3 ID = "rerank-multilingual-v3.0"
	CohereRerankEnV3    ID = "rerank-english-v3.0"
)

// CohereModels maps Cohere model IDs to their configurations.
var CohereModels = map[ID]Model{
	CommandAPlus: {
		ID:                    CommandAPlus,
		Name:                  "Command A+",
		Provider:              ProviderCohere,
		APIModel:              "command-a-plus-05-2026",
		CostPer1MIn:           2.50,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          10.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: true,
	},
	CommandA: {
		ID:                    CommandA,
		Name:                  "Command A",
		Provider:              ProviderCohere,
		APIModel:              "command-a-03-2025",
		CostPer1MIn:           2.50,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          10.00,
		ContextWindow:         256_000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: true,
	},
	CommandR7B: {
		ID:                    CommandR7B,
		Name:                  "Command R7B",
		Provider:              ProviderCohere,
		APIModel:              "command-r7b-12-2024",
		CostPer1MIn:           0.0375,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.15,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	CommandRPlus: {
		ID:                    CommandRPlus,
		Name:                  "Command R+",
		Provider:              ProviderCohere,
		APIModel:              "command-r-plus",
		CostPer1MIn:           2.50,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          10.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	CommandR: {
		ID:                    CommandR,
		Name:                  "Command R",
		Provider:              ProviderCohere,
		APIModel:              "command-r",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.60,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}

// CohereEmbeddingModels maps Cohere embedding model IDs to their configurations.
var CohereEmbeddingModels = map[ID]EmbeddingModel{
	CohereEmbedV4: {
		ID:              CohereEmbedV4,
		Name:            "Cohere Embed v4.0",
		Provider:        ProviderCohere,
		APIModel:        "embed-v4.0",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  128_000,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	CohereEmbedMultiV3: {
		ID:              CohereEmbedMultiV3,
		Name:            "Cohere Embed Multilingual v3.0",
		Provider:        ProviderCohere,
		APIModel:        "embed-multilingual-v3.0",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	CohereEmbedEnV3: {
		ID:              CohereEmbedEnV3,
		Name:            "Cohere Embed English v3.0",
		Provider:        ProviderCohere,
		APIModel:        "embed-english-v3.0",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	CohereEmbedMultiLightV3: {
		ID:              CohereEmbedMultiLightV3,
		Name:            "Cohere Embed Multilingual Light v3.0",
		Provider:        ProviderCohere,
		APIModel:        "embed-multilingual-light-v3.0",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  512,
		EmbeddingDims:   384,
		MaxBatchSize:    96,
	},
	CohereEmbedEnLightV3: {
		ID:              CohereEmbedEnLightV3,
		Name:            "Cohere Embed English Light v3.0",
		Provider:        ProviderCohere,
		APIModel:        "embed-english-light-v3.0",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  512,
		EmbeddingDims:   384,
		MaxBatchSize:    96,
	},
}

// CohereRerankerModels maps Cohere reranker model IDs to their configurations.
var CohereRerankerModels = map[ID]RerankerModel{
	CohereRerank4Pro: {
		ID:              CohereRerank4Pro,
		Name:            "Cohere Rerank v4.0 Pro",
		Provider:        ProviderCohere,
		APIModel:        "rerank-v4.0-pro",
		CostPer1MTokens: 2.50,
		MaxQueryTokens:  4096,
		MaxTotalTokens:  32_768,
	},
	CohereRerank4Fast: {
		ID:              CohereRerank4Fast,
		Name:            "Cohere Rerank v4.0 Fast",
		Provider:        ProviderCohere,
		APIModel:        "rerank-v4.0-fast",
		CostPer1MTokens: 2.00,
		MaxQueryTokens:  4096,
		MaxTotalTokens:  32_768,
	},
	CohereRerank35: {
		ID:              CohereRerank35,
		Name:            "Cohere Rerank v3.5",
		Provider:        ProviderCohere,
		APIModel:        "rerank-v3.5",
		CostPer1MTokens: 2.00,
		MaxQueryTokens:  2048,
		MaxTotalTokens:  4096,
	},
	CohereRerankMultiV3: {
		ID:              CohereRerankMultiV3,
		Name:            "Cohere Rerank Multilingual v3.0",
		Provider:        ProviderCohere,
		APIModel:        "rerank-multilingual-v3.0",
		CostPer1MTokens: 2.00,
		MaxQueryTokens:  2048,
		MaxTotalTokens:  4096,
	},
	CohereRerankEnV3: {
		ID:              CohereRerankEnV3,
		Name:            "Cohere Rerank English v3.0",
		Provider:        ProviderCohere,
		APIModel:        "rerank-english-v3.0",
		CostPer1MTokens: 2.00,
		MaxQueryTokens:  2048,
		MaxTotalTokens:  4096,
	},
}
