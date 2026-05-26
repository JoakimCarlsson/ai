package model

// Gemini provider plus Gemini and Imagen model IDs for this registry.
const (
	ProviderGemini Provider = "gemini"

	// gemini 3.5
	Gemini35Flash ID = "gemini-3.5-flash"

	// gemini 3.1
	Gemini31FlashLitePreview  ID = "gemini-3.1-flash-lite-preview"
	Gemini31FlashLite         ID = "gemini-3.1-flash-lite"
	Gemini31ProPreview        ID = "gemini-3.1-pro-preview"
	Gemini31FlashImagePreview ID = "gemini-3.1-flash-image-preview"
	Gemini31FlashTTSPreview   ID = "gemini-3.1-flash-tts-preview"
	Gemini31FlashLivePreview  ID = "gemini-3.1-flash-live-preview"

	// gemini 3
	Gemini3Pro          ID = "gemini-3-pro"
	Gemini3Flash        ID = "gemini-3-flash"
	Gemini3FlashPreview ID = "gemini-3-flash-preview"
	Gemini31Pro         ID = "gemini-3.1-pro"
	Gemini3ProImage     ID = "gemini-3-pro-image"

	// gemini 2.5
	Gemini25Flash              ID = "gemini-2.5-flash"
	Gemini25FlashLite          ID = "gemini-2.5-flash-lite"
	Gemini25                   ID = "gemini-2.5"
	Gemini25FlashImage         ID = "gemini-2.5-flash-image"
	Gemini25FlashLitePreview   ID = "gemini-2.5-flash-lite-preview-09-2025"
	Gemini25FlashNativeAudio   ID = "gemini-2.5-flash-native-audio-preview-12-2025"
	Gemini25FlashPreviewTTS    ID = "gemini-2.5-flash-preview-tts"
	Gemini25ProPreviewTTS      ID = "gemini-2.5-pro-preview-tts"
	Gemini25ComputerUsePreview ID = "gemini-2.5-computer-use-preview-10-2025"

	// gemini 2.0
	Gemini20Flash     ID = "gemini-2.0-flash"
	Gemini20FlashLite ID = "gemini-2.0-flash-lite"

	// imagen
	Imagen4      ID = "imagen-4.0"
	Imagen4Ultra ID = "imagen-4.0-ultra"
	Imagen4Fast  ID = "imagen-4.0-fast"

	// imbeddings
	GeminiTextEmbedding004 ID = "text-embedding-004"
	GeminiEmbedding2       ID = "gemini-embedding-2"

	// NanoBanana2 is an alias for Gemini31FlashImagePreview
	NanoBanana2 ID = Gemini31FlashImagePreview
)

// GeminiModels maps Gemini chat model IDs to their configurations.
var GeminiModels = map[ID]Model{
	Gemini35Flash: {
		ID:                    Gemini35Flash,
		Name:                  "Gemini 3.5 Flash",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3.5-flash",
		CostPer1MIn:           1.50,
		CostPer1MInCached:     0.15,
		CostPer1MOutCached:    0,
		CostPer1MOut:          9.00,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31FlashLite: {
		ID:                    Gemini31FlashLite,
		Name:                  "Gemini 3.1 Flash Lite",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3.1-flash-lite",
		CostPer1MIn:           0.25,
		CostPer1MInCached:     0.025,
		CostPer1MOutCached:    0,
		CostPer1MOut:          1.50,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31ProPreview: {
		ID:                    Gemini31ProPreview,
		Name:                  "Gemini 3.1 Pro Preview",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3.1-pro-preview",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0.20,
		CostPer1MOutCached:    0,
		CostPer1MOut:          12.00,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31FlashLitePreview: {
		ID:                    Gemini31FlashLitePreview,
		Name:                  "Gemini 3.1 Flash Lite Preview",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3.1-flash-lite-preview",
		CostPer1MIn:           0.25,
		CostPer1MInCached:     0.025,
		CostPer1MOutCached:    0,
		CostPer1MOut:          1.50,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31FlashLivePreview: {
		ID:                    Gemini31FlashLivePreview,
		Name:                  "Gemini 3.1 Flash Live Preview",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3.1-flash-live-preview",
		CostPer1MIn:           0.75,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          4.50,
		ContextWindow:         131072,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: false,
	},
	Gemini3Pro: {
		ID:                    Gemini3Pro,
		Name:                  "Gemini 3 Pro",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3-pro",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0.20,
		CostPer1MOutCached:    0,
		CostPer1MOut:          12.00,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini3Flash: {
		ID:                    Gemini3Flash,
		Name:                  "Gemini 3 Flash",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3-flash-preview",
		CostPer1MIn:           0.50,
		CostPer1MInCached:     0.05,
		CostPer1MOutCached:    0,
		CostPer1MOut:          3.00,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini3FlashPreview: {
		ID:                    Gemini3FlashPreview,
		Name:                  "Gemini 3 Flash Preview",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3-flash-preview",
		CostPer1MIn:           0.50,
		CostPer1MInCached:     0.05,
		CostPer1MOutCached:    0,
		CostPer1MOut:          3.00,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31Pro: {
		ID:                    Gemini31Pro,
		Name:                  "Gemini 3.1 Pro",
		Provider:              ProviderGemini,
		APIModel:              "gemini-3.1-pro-preview",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0.20,
		CostPer1MOutCached:    0,
		CostPer1MOut:          12.00,
		ContextWindow:         1_048_576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25Flash: {
		ID:                    Gemini25Flash,
		Name:                  "Gemini 2.5 Flash",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.5-flash",
		CostPer1MIn:           0.30,
		CostPer1MInCached:     0.03,
		CostPer1MOutCached:    0,
		CostPer1MOut:          2.50,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25FlashLite: {
		ID:                    Gemini25FlashLite,
		Name:                  "Gemini 2.5 Flash Lite",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.5-flash-lite",
		CostPer1MIn:           0.10,
		CostPer1MInCached:     0.01,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.40,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25: {
		ID:                    Gemini25,
		Name:                  "Gemini 2.5 Pro",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.5-pro",
		CostPer1MIn:           1.25,
		CostPer1MInCached:     0.125,
		CostPer1MOutCached:    0,
		CostPer1MOut:          10.00,
		ContextWindow:         2000000,
		DefaultMaxTokens:      64000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25FlashLitePreview: {
		ID:                    Gemini25FlashLitePreview,
		Name:                  "Gemini 2.5 Flash Lite Preview",
		Provider:              ProviderGemini,
		APIModel:              "gemini-2.5-flash-lite-preview-09-2025",
		CostPer1MIn:           0.10,
		CostPer1MInCached:     0.01,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.40,
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
		CostPer1MInCached:     0.025,
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
		CostPer1MIn:           0.075,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.30,
		ContextWindow:         1000000,
		DefaultMaxTokens:      6000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}

// GeminiImageGenerationModels maps Gemini and Imagen image-generation model IDs to their configurations.
var GeminiImageGenerationModels = map[ID]ImageGenerationModel{
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
		MaxPromptTokens: 4000,
		SupportedAspectRatios: []string{
			"1:1",
			"2:3",
			"3:2",
			"3:4",
			"4:3",
			"4:5",
			"5:4",
			"9:16",
			"16:9",
			"21:9",
		},
		DefaultAspectRatio: "1:1",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
	Gemini3ProImage: {
		ID:       Gemini3ProImage,
		Name:     "Gemini 3 Pro Image (Nano Banana Pro)",
		Provider: ProviderGemini,
		APIModel: "gemini-3-pro-image-preview",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.134,
			},
			"3:4": {
				"default": 0.134,
			},
			"4:3": {
				"default": 0.134,
			},
			"9:16": {
				"default": 0.134,
			},
			"16:9": {
				"default": 0.134,
			},
		},
		MaxPromptTokens:       65536,
		SupportedAspectRatios: []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultAspectRatio:    "1:1",
		SupportedQualities:    []string{"default"},
		DefaultQuality:        "default",
	},
	Gemini31FlashImagePreview: {
		ID:       Gemini31FlashImagePreview,
		Name:     "Gemini 3.1 Flash Image Preview (Nano Banana 2)",
		Provider: ProviderGemini,
		APIModel: "gemini-3.1-flash-image-preview",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.067,
			},
			"3:4": {
				"default": 0.067,
			},
			"4:3": {
				"default": 0.067,
			},
			"9:16": {
				"default": 0.067,
			},
			"16:9": {
				"default": 0.067,
			},
			"1:4": {
				"default": 0.067,
			},
			"4:1": {
				"default": 0.067,
			},
			"1:8": {
				"default": 0.067,
			},
			"8:1": {
				"default": 0.067,
			},
		},
		MaxPromptTokens: 131072,
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
			"1:4",
			"4:1",
			"1:8",
			"8:1",
		},
		DefaultAspectRatio: "1:1",
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
		MaxPromptTokens:       4000,
		SupportedAspectRatios: []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultAspectRatio:    "1:1",
		SupportedQualities:    []string{"default"},
		DefaultQuality:        "default",
	},
	Imagen4Ultra: {
		ID:       Imagen4Ultra,
		Name:     "Imagen 4 Ultra",
		Provider: ProviderGemini,
		APIModel: "imagen-4.0-ultra-generate-001",
		Pricing: map[string]map[string]float64{
			"1:1": {
				"default": 0.06,
			},
			"3:4": {
				"default": 0.06,
			},
			"4:3": {
				"default": 0.06,
			},
			"9:16": {
				"default": 0.06,
			},
			"16:9": {
				"default": 0.06,
			},
		},
		MaxPromptTokens:       4000,
		SupportedAspectRatios: []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultAspectRatio:    "1:1",
		SupportedQualities:    []string{"default"},
		DefaultQuality:        "default",
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
		MaxPromptTokens:       4000,
		SupportedAspectRatios: []string{"1:1", "3:4", "4:3", "9:16", "16:9"},
		DefaultAspectRatio:    "1:1",
		SupportedQualities:    []string{"default"},
		DefaultQuality:        "default",
	},
}

// GeminiEmbeddingModels maps Gemini embedding model IDs to their configurations.
var GeminiEmbeddingModels = map[ID]EmbeddingModel{
	GeminiTextEmbedding004: {
		ID:                  GeminiTextEmbedding004,
		Name:                "Gemini Text Embedding 004",
		Provider:            ProviderGemini,
		APIModel:            "text-embedding-004",
		CostPer1MTokens:     0.15,
		MaxInputTokens:      2048,
		EmbeddingDims:       768,
		SupportedDimensions: []int{768, 512, 256},
		MaxBatchSize:        100,
	},
	GeminiEmbedding2: {
		ID:              GeminiEmbedding2,
		Name:            "Gemini Embedding 2",
		Provider:        ProviderGemini,
		APIModel:        "gemini-embedding-2",
		CostPer1MTokens: 0.20,
		MaxInputTokens:  8192,
		EmbeddingDims:   768,
		SupportedDimensions: []int{
			768,
			1536,
			3072,
		}, // flexible. supports: 128 - 3072,
		MaxBatchSize: 100,
	},
}
