package model

const (
	ProviderGemini ModelProvider = "gemini"

	Gemini25Flash     ModelID = "gemini-2.5-flash"
	Gemini25          ModelID = "gemini-2.5"
	Gemini20Flash     ModelID = "gemini-2.0-flash"
	Gemini20FlashLite ModelID = "gemini-2.0-flash-lite"

	Gemini25FlashImage ModelID = "gemini-2.5-flash-image"
	Imagen3            ModelID = "imagen-3.0"
	Imagen4            ModelID = "imagen-4.0"
	Imagen4Ultra       ModelID = "imagen-4.0-ultra"
	Imagen4Fast        ModelID = "imagen-4.0-fast"
)

var GeminiModels = map[ModelID]Model{
	Gemini25Flash: {
		ID:                    Gemini25Flash,
		Name:                  "Gemini 2.5 Flash",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.5-flash-preview-04-17",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.60,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25: {
		ID:                    Gemini25,
		Name:                  "Gemini 2.5 Pro",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.5-pro-preview-03-25",
		CostPer1MIn:           1.25,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          10,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},

	Gemini20Flash: {
		ID:                    Gemini20Flash,
		Name:                  "Gemini 2.0 Flash",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.0-flash",
		CostPer1MIn:           0.10,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.40,
		ContextWindow:         1000000,
		DefaultMaxTokens:      6000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini20FlashLite: {
		ID:                    Gemini20FlashLite,
		Name:                  "Gemini 2.0 Flash Lite",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.0-flash-lite",
		CostPer1MIn:           0.05,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.30,
		ContextWindow:         1000000,
		DefaultMaxTokens:      6000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}

var GeminiImageGenerationModels = map[ModelID]ImageGenerationModel{
	Gemini25FlashImage: {
		ID:       Gemini25FlashImage,
		Name:     "Gemini 2.5 Flash Image",
		Provider: ProviderGemini,
		APIModel: "gemini-2.5-flash-image",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.039,
			},
			"3:4": {
				"default": 0.039,
			},
			"4:3": {
				"default": 0.039,
			},
			"9:16": {
				"default": 0.039,
			},
			"16:9": {
				"default": 0.039,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultSize:        "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
	Imagen3: {
		ID:       Imagen3,
		Name:     "Imagen 3",
		Provider: ProviderGemini,
		APIModel: "imagen-3.0-generate-002",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.03,
			},
			"3:4": {
				"default": 0.03,
			},
			"4:3": {
				"default": 0.03,
			},
			"9:16": {
				"default": 0.03,
			},
			"16:9": {
				"default": 0.03,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultSize:        "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
	Imagen4: {
		ID:       Imagen4,
		Name:     "Imagen 4",
		Provider: ProviderGemini,
		APIModel: "imagen-4.0-generate-001",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.04,
			},
			"3:4": {
				"default": 0.04,
			},
			"4:3": {
				"default": 0.04,
			},
			"9:16": {
				"default": 0.04,
			},
			"16:9": {
				"default": 0.04,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultSize:        "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
	Imagen4Ultra: {
		ID:       Imagen4Ultra,
		Name:     "Imagen 4 Ultra",
		Provider: ProviderGemini,
		APIModel: "imagen-4.0-ultra-generate-001",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.04,
			},
			"3:4": {
				"default": 0.04,
			},
			"4:3": {
				"default": 0.04,
			},
			"9:16": {
				"default": 0.04,
			},
			"16:9": {
				"default": 0.04,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultSize:        "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
	Imagen4Fast: {
		ID:       Imagen4Fast,
		Name:     "Imagen 4 Fast",
		Provider: ProviderGemini,
		APIModel: "imagen-4.0-fast-generate-001",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.02,
			},
			"3:4": {
				"default": 0.02,
			},
			"4:3": {
				"default": 0.02,
			},
			"9:16": {
				"default": 0.02,
			},
			"16:9": {
				"default": 0.02,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultSize:        "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
}
