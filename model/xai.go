package model

const (
	ProviderXAI ModelProvider = "xai"

	XAIGrok4                  ModelID = "grok-4-0709"
	XAIGrok4FastReasoning     ModelID = "grok-4-fast-reasoning"
	XAIGrok4FastNonReasoning  ModelID = "grok-4-fast-non-reasoning"
	XAIGrok41FastReasoning    ModelID = "grok-4-1-fast-reasoning"
	XAIGrok41FastNonReasoning ModelID = "grok-4-1-fast-non-reasoning"
	XAIGrok3                  ModelID = "grok-3"
	XAIGrok3Mini              ModelID = "grok-3-mini"
	XAIGrok3Fast              ModelID = "grok-3-fast"
	XAIGrok3MiniFast          ModelID = "grok-3-mini-fast"
	XAIGrok2Vision            ModelID = "grok-2-vision-1212"
	XAIGrokCodeFast1          ModelID = "grok-code-fast-1"
	XAIGrok2Image             ModelID = "grok-2-image-1212"
)

var XAIModels = map[ModelID]Model{
	XAIGrok4: {
		ID:                    XAIGrok4,
		Name:                  "Grok4",
		Provider:              ProviderXAI,
		APIModel:              "grok-4-0709",
		CostPer1MIn:           3.0,
		CostPer1MInCached:     0.75,
		CostPer1MOut:          15.0,
		CostPer1MOutCached:    0,
		ContextWindow:         256_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok3: {
		ID:                    XAIGrok3,
		Name:                  "Grok3",
		Provider:              ProviderXAI,
		APIModel:              "grok-3",
		CostPer1MIn:           3.0,
		CostPer1MInCached:     0,
		CostPer1MOut:          15.0,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok3Mini: {
		ID:                    XAIGrok3Mini,
		Name:                  "Grok3 Mini",
		Provider:              ProviderXAI,
		APIModel:              "grok-3-mini",
		CostPer1MIn:           0.3,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.5,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok3Fast: {
		ID:                    XAIGrok3Fast,
		Name:                  "Grok3 Fast",
		Provider:              ProviderXAI,
		APIModel:              "grok-3-fast",
		CostPer1MIn:           5.0,
		CostPer1MInCached:     0,
		CostPer1MOut:          25.0,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok3MiniFast: {
		ID:                    XAIGrok3MiniFast,
		Name:                  "Grok3 Mini Fast",
		Provider:              ProviderXAI,
		APIModel:              "grok-3-mini-fast",
		CostPer1MIn:           0.6,
		CostPer1MInCached:     0,
		CostPer1MOut:          4.0,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok2Vision: {
		ID:                    XAIGrok2Vision,
		Name:                  "Grok2 Vision",
		Provider:              ProviderXAI,
		APIModel:              "grok-2-vision-1212",
		CostPer1MIn:           2.0,
		CostPer1MInCached:     0,
		CostPer1MOut:          10.0,
		CostPer1MOutCached:    0,
		ContextWindow:         32_768,
		DefaultMaxTokens:      4_000,
		SupportsStructuredOut: true,
	},
	XAIGrok4FastReasoning: {
		ID:                    XAIGrok4FastReasoning,
		Name:                  "Grok4 Fast Reasoning",
		Provider:              ProviderXAI,
		APIModel:              "grok-4-fast-reasoning",
		CostPer1MIn:           0.20,
		CostPer1MInCached:     0.05,
		CostPer1MOut:          0.50,
		CostPer1MOutCached:    0,
		ContextWindow:         2_000_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok4FastNonReasoning: {
		ID:                    XAIGrok4FastNonReasoning,
		Name:                  "Grok4 Fast Non-Reasoning",
		Provider:              ProviderXAI,
		APIModel:              "grok-4-fast-non-reasoning",
		CostPer1MIn:           0.20,
		CostPer1MInCached:     0.05,
		CostPer1MOut:          0.50,
		CostPer1MOutCached:    0,
		ContextWindow:         2_000_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok41FastReasoning: {
		ID:                    XAIGrok41FastReasoning,
		Name:                  "Grok4.1 Fast Reasoning",
		Provider:              ProviderXAI,
		APIModel:              "grok-4-1-fast-reasoning",
		CostPer1MIn:           0.20,
		CostPer1MInCached:     0.05,
		CostPer1MOut:          0.50,
		CostPer1MOutCached:    0,
		ContextWindow:         2_000_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok41FastNonReasoning: {
		ID:                    XAIGrok41FastNonReasoning,
		Name:                  "Grok4.1 Fast Non-Reasoning",
		Provider:              ProviderXAI,
		APIModel:              "grok-4-1-fast-non-reasoning",
		CostPer1MIn:           0.20,
		CostPer1MInCached:     0.05,
		CostPer1MOut:          0.50,
		CostPer1MOutCached:    0,
		ContextWindow:         2_000_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrokCodeFast1: {
		ID:                    XAIGrokCodeFast1,
		Name:                  "Grok Code Fast 1",
		Provider:              ProviderXAI,
		APIModel:              "grok-code-fast-1",
		CostPer1MIn:           0.20,
		CostPer1MInCached:     0.02,
		CostPer1MOut:          1.50,
		CostPer1MOutCached:    0,
		ContextWindow:         256_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
}

var XAIImageGenerationModels = map[ModelID]ImageGenerationModel{
	XAIGrok2Image: {
		ID:       XAIGrok2Image,
		Name:     "Grok 2 Image",
		Provider: ProviderXAI,
		APIModel: "grok-2-image-1212",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.07,
			},
		},
		MaxPromptTokens:    1000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
}
