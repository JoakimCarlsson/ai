package model

// xAI provider identifier and Grok model IDs for this registry.
const (
	ProviderXAI Provider = "xai"

	XAIGrok4                  ID = "grok-4-0709"
	XAIGrok4FastReasoning     ID = "grok-4-fast-reasoning"
	XAIGrok4FastNonReasoning  ID = "grok-4-fast-non-reasoning"
	XAIGrok41FastReasoning    ID = "grok-4-1-fast-reasoning"
	XAIGrok41FastNonReasoning ID = "grok-4-1-fast-non-reasoning"
	XAIGrok3                  ID = "grok-3"
	XAIGrok3Mini              ID = "grok-3-mini"
	XAIGrok3Fast              ID = "grok-3-fast"
	XAIGrok3MiniFast          ID = "grok-3-mini-fast"
	XAIGrok2Vision            ID = "grok-2-vision-1212"
	XAIGrokCodeFast1          ID = "grok-code-fast-1"
	XAIGrok420Reasoning       ID = "grok-4.20-0309-reasoning"
	XAIGrok420NonReasoning    ID = "grok-4.20-0309-non-reasoning"
	XAIGrok420MultiAgent      ID = "grok-4.20-multi-agent-0309"
	XAIGrok43                 ID = "grok-4.3"
	XAIGrok45                 ID = "grok-4.5"
	XAIGrokBuild01            ID = "grok-build-0.1"
	XAIGrok2Image             ID = "grok-2-image-1212"
	XAIGrokImagineImage       ID = "grok-imagine-image"
	XAIGrokImagineImagePro    ID = "grok-imagine-image-pro"
)

// XAIModels maps xAI chat model IDs to their configurations.
var XAIModels = map[ID]Model{
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
	XAIGrok420Reasoning: {
		ID:                    XAIGrok420Reasoning,
		Name:                  "Grok 4.20 Reasoning",
		Provider:              ProviderXAI,
		APIModel:              "grok-4.20-0309-reasoning",
		CostPer1MIn:           2.0,
		CostPer1MInCached:     0,
		CostPer1MOut:          6.0,
		CostPer1MOutCached:    0,
		ContextWindow:         2_000_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok420NonReasoning: {
		ID:                    XAIGrok420NonReasoning,
		Name:                  "Grok 4.20 Non-Reasoning",
		Provider:              ProviderXAI,
		APIModel:              "grok-4.20-0309-non-reasoning",
		CostPer1MIn:           2.0,
		CostPer1MInCached:     0,
		CostPer1MOut:          6.0,
		CostPer1MOutCached:    0,
		ContextWindow:         2_000_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	XAIGrok420MultiAgent: {
		ID:                    XAIGrok420MultiAgent,
		Name:                  "Grok 4.20 Multi-Agent",
		Provider:              ProviderXAI,
		APIModel:              "grok-4.20-multi-agent-0309",
		CostPer1MIn:           2.0,
		CostPer1MInCached:     0,
		CostPer1MOut:          6.0,
		CostPer1MOutCached:    0,
		ContextWindow:         2_000_000,
		DefaultMaxTokens:      20_000,
		SupportsStructuredOut: true,
	},
	// Pricing source: https://docs.x.ai/developers/models/grok-4.3. Fetched: 2026-05-04.
	// Reasoning is enabled by default; reasoning tokens are billed at the
	// output rate. Per OpenRouter, requests > 200k tokens are billed at a
	// higher tier; the rate below is the base tier.
	XAIGrok43: {
		ID:                    XAIGrok43,
		Name:                  "Grok 4.3",
		Provider:              ProviderXAI,
		APIModel:              "grok-4.3",
		CostPer1MIn:           1.25,
		CostPer1MInCached:     0,
		CostPer1MOut:          2.50,
		CostPer1MOutCached:    0,
		ContextWindow:         1_000_000,
		DefaultMaxTokens:      32_000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	// Pricing source: https://docs.x.ai/developers/models/grok-4.5. Fetched: 2026-07-10.
	// Flagship general-intelligence model; supports reasoning and non-reasoning
	// modes. Tiered pricing applies above a 200k-token prompt threshold; the
	// rate below is the base (<=200k) tier.
	XAIGrok45: {
		ID:                    XAIGrok45,
		Name:                  "Grok 4.5",
		Provider:              ProviderXAI,
		APIModel:              "grok-4.5",
		CostPer1MIn:           2.0,
		CostPer1MInCached:     0.50,
		CostPer1MOut:          6.0,
		CostPer1MOutCached:    0,
		ContextWindow:         500_000,
		DefaultMaxTokens:      32_000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	// Pricing source: https://docs.x.ai/developers/models/grok-build-0.1. Fetched: 2026-07-10.
	// Fast coding model for agentic software-engineering workflows.
	XAIGrokBuild01: {
		ID:                    XAIGrokBuild01,
		Name:                  "Grok Build 0.1",
		Provider:              ProviderXAI,
		APIModel:              "grok-build-0.1",
		CostPer1MIn:           1.0,
		CostPer1MInCached:     0.20,
		CostPer1MOut:          2.0,
		CostPer1MOutCached:    0,
		ContextWindow:         256_000,
		DefaultMaxTokens:      20_000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}

// XAIImageGenerationModels maps xAI image generation model IDs to their configurations.
//
// Pricing source: https://docs.x.ai/developers/models/. Fetched: 2026-05-04.
// Grok Imagine pricing is flat per image regardless of resolution or quality.
var XAIImageGenerationModels = map[ID]ImageGenerationModel{
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
	XAIGrokImagineImage: {
		ID:       XAIGrokImagineImage,
		Name:     "Grok Imagine Image",
		Provider: ProviderXAI,
		APIModel: "grok-imagine-image",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.02,
			},
		},
		MaxPromptTokens: 1000,
		SupportedAspectRatios: []string{
			"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "2:1", "1:2",
			"19.5:9", "9:19.5", "20:9", "9:20", "auto",
		},
		DefaultAspectRatio: "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
	XAIGrokImagineImagePro: {
		ID:       XAIGrokImagineImagePro,
		Name:     "Grok Imagine Image Pro",
		Provider: ProviderXAI,
		APIModel: "grok-imagine-image-pro",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.07,
			},
		},
		MaxPromptTokens: 1000,
		SupportedAspectRatios: []string{
			"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "2:1", "1:2",
			"19.5:9", "9:19.5", "20:9", "9:20", "auto",
		},
		DefaultAspectRatio: "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
}
