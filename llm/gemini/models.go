package gemini

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Gemini provider plus Gemini and Imagen model IDs for this registry.
const (
	Gemini36Flash            string = "gemini-3.6-flash"
	Gemini35Flash            string = "gemini-3.5-flash"
	Gemini35FlashLite        string = "gemini-3.5-flash-lite"
	Gemini31FlashLitePreview string = "gemini-3.1-flash-lite-preview"
	Gemini31FlashLite        string = "gemini-3.1-flash-lite"
	Gemini31ProPreview       string = "gemini-3.1-pro-preview"
	Gemini31FlashLivePreview string = "gemini-3.1-flash-live-preview"
	Gemini3Pro               string = "gemini-3-pro"
	Gemini3Flash             string = "gemini-3-flash"
	Gemini3FlashPreview      string = "gemini-3-flash-preview"
	Gemini31Pro              string = "gemini-3.1-pro"
	Gemini25Flash            string = "gemini-2.5-flash"
	Gemini25FlashLite        string = "gemini-2.5-flash-lite"
	Gemini25                 string = "gemini-2.5"
	Gemini25FlashLitePreview string = "gemini-2.5-flash-lite-preview-09-2025"
	Gemini20Flash            string = "gemini-2.0-flash"
	Gemini20FlashLite        string = "gemini-2.0-flash-lite"
)

// Gemini31FlashTTSPreview is a published model ID with no catalog entry.
const Gemini31FlashTTSPreview string = "gemini-3.1-flash-tts-preview"

// Gemini25FlashNativeAudio is a published model ID with no catalog entry.
const Gemini25FlashNativeAudio string = "gemini-2.5-flash-native-audio-preview-12-2025"

// Gemini25FlashPreviewTTS is a published model ID with no catalog entry.
const Gemini25FlashPreviewTTS string = "gemini-2.5-flash-preview-tts"

// Gemini25ProPreviewTTS is a published model ID with no catalog entry.
const Gemini25ProPreviewTTS string = "gemini-2.5-pro-preview-tts"

// Gemini25ComputerUsePreview is a published model ID with no catalog entry.
const Gemini25ComputerUsePreview string = "gemini-2.5-computer-use-preview-10-2025"

// Models maps Gemini chat model IDs to their configurations.
//
// Pricing source: https://ai.google.dev/gemini-api/docs/pricing.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Gemini36Flash: {
		ID:                    Gemini36Flash,
		Name:                  "Gemini 3.6 Flash",
		Provider:              "gemini",
		APIModel:              "gemini-3.6-flash",
		CostPer1MIn:           1.5,
		CostPer1MOut:          7.5,
		CostPer1MInCached:     0.15,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini35FlashLite: {
		ID:                    Gemini35FlashLite,
		Name:                  "Gemini 3.5 Flash Lite",
		Provider:              "gemini",
		APIModel:              "gemini-3.5-flash-lite",
		CostPer1MIn:           0.3,
		CostPer1MOut:          2.5,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini35Flash: {
		ID:                    Gemini35Flash,
		Name:                  "Gemini 3.5 Flash",
		Provider:              "gemini",
		APIModel:              "gemini-3.5-flash",
		CostPer1MIn:           1.5,
		CostPer1MOut:          9,
		CostPer1MInCached:     0.15,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31FlashLite: {
		ID:                    Gemini31FlashLite,
		Name:                  "Gemini 3.1 Flash Lite",
		Provider:              "gemini",
		APIModel:              "gemini-3.1-flash-lite",
		CostPer1MIn:           0.25,
		CostPer1MOut:          1.5,
		CostPer1MInCached:     0.025,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31ProPreview: {
		ID:                    Gemini31ProPreview,
		Name:                  "Gemini 3.1 Pro Preview",
		Provider:              "gemini",
		APIModel:              "gemini-3.1-pro-preview",
		CostPer1MIn:           2,
		CostPer1MOut:          12,
		CostPer1MInCached:     0.2,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31FlashLitePreview: {
		ID:                    Gemini31FlashLitePreview,
		Name:                  "Gemini 3.1 Flash Lite Preview",
		Provider:              "gemini",
		APIModel:              "gemini-3.1-flash-lite-preview",
		CostPer1MIn:           0.25,
		CostPer1MOut:          1.5,
		CostPer1MInCached:     0.025,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31FlashLivePreview: {
		ID:                  Gemini31FlashLivePreview,
		Name:                "Gemini 3.1 Flash Live Preview",
		Provider:            "gemini",
		APIModel:            "gemini-3.1-flash-live-preview",
		CostPer1MIn:         0.75,
		CostPer1MOut:        4.5,
		ContextWindow:       131072,
		DefaultMaxTokens:    65536,
		CanReason:           true,
		SupportsAttachments: true,
	},
	Gemini3Pro: {
		ID:                    Gemini3Pro,
		Name:                  "Gemini 3 Pro",
		Provider:              "gemini",
		APIModel:              "gemini-3-pro",
		CostPer1MIn:           2,
		CostPer1MOut:          12,
		CostPer1MInCached:     0.2,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini3Flash: {
		ID:                    Gemini3Flash,
		Name:                  "Gemini 3 Flash",
		Provider:              "gemini",
		APIModel:              "gemini-3-flash-preview",
		CostPer1MIn:           0.5,
		CostPer1MOut:          3,
		CostPer1MInCached:     0.05,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini3FlashPreview: {
		ID:                    Gemini3FlashPreview,
		Name:                  "Gemini 3 Flash Preview",
		Provider:              "gemini",
		APIModel:              "gemini-3-flash-preview",
		CostPer1MIn:           0.5,
		CostPer1MOut:          3,
		CostPer1MInCached:     0.05,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31Pro: {
		ID:                    Gemini31Pro,
		Name:                  "Gemini 3.1 Pro",
		Provider:              "gemini",
		APIModel:              "gemini-3.1-pro-preview",
		CostPer1MIn:           2,
		CostPer1MOut:          12,
		CostPer1MInCached:     0.2,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25Flash: {
		ID:                    Gemini25Flash,
		Name:                  "Gemini 2.5 Flash",
		Provider:              "gemini",
		APIModel:              "gemini-2.5-flash",
		CostPer1MIn:           0.3,
		CostPer1MOut:          2.5,
		CostPer1MInCached:     0.03,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25FlashLite: {
		ID:                    Gemini25FlashLite,
		Name:                  "Gemini 2.5 Flash Lite",
		Provider:              "gemini",
		APIModel:              "gemini-2.5-flash-lite",
		CostPer1MIn:           0.1,
		CostPer1MOut:          0.4,
		CostPer1MInCached:     0.01,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25: {
		ID:                    Gemini25,
		Name:                  "Gemini 2.5 Pro",
		Provider:              "gemini",
		APIModel:              "gemini-2.5-pro",
		CostPer1MIn:           1.25,
		CostPer1MOut:          10,
		CostPer1MInCached:     0.125,
		ContextWindow:         2000000,
		DefaultMaxTokens:      64000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25FlashLitePreview: {
		ID:                    Gemini25FlashLitePreview,
		Name:                  "Gemini 2.5 Flash Lite Preview",
		Provider:              "gemini",
		APIModel:              "gemini-2.5-flash-lite-preview-09-2025",
		CostPer1MIn:           0.1,
		CostPer1MOut:          0.4,
		CostPer1MInCached:     0.01,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini20Flash: {
		ID:                    Gemini20Flash,
		Name:                  "Gemini 2.0 Flash",
		Provider:              "gemini",
		APIModel:              "gemini-2.0-flash",
		CostPer1MIn:           0.1,
		CostPer1MOut:          0.4,
		CostPer1MInCached:     0.025,
		ContextWindow:         1000000,
		DefaultMaxTokens:      6000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini20FlashLite: {
		ID:                    Gemini20FlashLite,
		Name:                  "Gemini 2.0 Flash Lite",
		Provider:              "gemini",
		APIModel:              "gemini-2.0-flash-lite",
		CostPer1MIn:           0.075,
		CostPer1MOut:          0.3,
		ContextWindow:         1000000,
		DefaultMaxTokens:      6000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}
