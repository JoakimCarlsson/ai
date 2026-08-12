package vertexai

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Vertex AI provider identifier and Gemini model IDs for this registry.
const (
	Gemini25Flash     string = "vertexai.gemini-2.5-flash"
	Gemini25          string = "vertexai.gemini-2.5"
	Gemini36Flash     string = "vertexai.gemini-3.6-flash"
	Gemini35Flash     string = "vertexai.gemini-3.5-flash"
	Gemini35FlashLite string = "vertexai.gemini-3.5-flash-lite"
	Gemini31Pro       string = "vertexai.gemini-3.1-pro"
	Gemini31FlashLite string = "vertexai.gemini-3.1-flash-lite"
	Gemini3Pro        string = "vertexai.gemini-3-pro"
	Gemini25FlashLite string = "vertexai.gemini-2.5-flash-lite"
	Gemini20Flash     string = "vertexai.gemini-2.0-flash"
	Gemini20FlashLite string = "vertexai.gemini-2.0-flash-lite"
)

// Models maps Vertex AI Gemini model IDs to their configurations.
//
// Pricing source: rates match the equivalent Gemini models; IDs from
// https://cloud.google.com/vertex-ai/generative-ai/docs/models.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Gemini25Flash: {
		ID:                    Gemini25Flash,
		Name:                  "VertexAI: Gemini 2.5 Flash",
		Provider:              "vertexai",
		APIModel:              "gemini-2.5-flash-preview-04-17",
		Currency:              "USD",
		CostPer1MIn:           0.3,
		CostPer1MOut:          2.5,
		CostPer1MInCached:     0.03,
		ContextWindow:         1000000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25: {
		ID:                    Gemini25,
		Name:                  "VertexAI: Gemini 2.5 Pro",
		Provider:              "vertexai",
		APIModel:              "gemini-2.5-pro-preview-03-25",
		Currency:              "USD",
		CostPer1MIn:           1.25,
		CostPer1MOut:          10,
		CostPer1MInCached:     0.125,
		ContextWindow:         2000000,
		DefaultMaxTokens:      64000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini35Flash: {
		ID:                    Gemini35Flash,
		Name:                  "VertexAI: Gemini 3.5 Flash",
		Provider:              "vertexai",
		APIModel:              "gemini-3.5-flash",
		Currency:              "USD",
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
		Name:                  "VertexAI: Gemini 3.1 Flash Lite",
		Provider:              "vertexai",
		APIModel:              "gemini-3.1-flash-lite",
		Currency:              "USD",
		CostPer1MIn:           0.25,
		CostPer1MOut:          1.5,
		CostPer1MInCached:     0.025,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini3Pro: {
		ID:                    Gemini3Pro,
		Name:                  "VertexAI: Gemini 3 Pro",
		Provider:              "vertexai",
		APIModel:              "gemini-3-pro",
		Currency:              "USD",
		CostPer1MIn:           2,
		CostPer1MOut:          12,
		CostPer1MInCached:     0.2,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini25FlashLite: {
		ID:                    Gemini25FlashLite,
		Name:                  "VertexAI: Gemini 2.5 Flash Lite",
		Provider:              "vertexai",
		APIModel:              "gemini-2.5-flash-lite",
		Currency:              "USD",
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
		Name:                  "VertexAI: Gemini 2.0 Flash",
		Provider:              "vertexai",
		APIModel:              "gemini-2.0-flash",
		Currency:              "USD",
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
		Name:                  "VertexAI: Gemini 2.0 Flash Lite",
		Provider:              "vertexai",
		APIModel:              "gemini-2.0-flash-lite",
		Currency:              "USD",
		CostPer1MIn:           0.075,
		CostPer1MOut:          0.3,
		ContextWindow:         1000000,
		DefaultMaxTokens:      6000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini36Flash: {
		ID:                    Gemini36Flash,
		Name:                  "VertexAI: Gemini 3.6 Flash",
		Provider:              "vertexai",
		APIModel:              "gemini-3.6-flash",
		Currency:              "USD",
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
		Name:                  "VertexAI: Gemini 3.5 Flash Lite",
		Provider:              "vertexai",
		APIModel:              "gemini-3.5-flash-lite",
		Currency:              "USD",
		CostPer1MIn:           0.3,
		CostPer1MOut:          2.5,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemini31Pro: {
		ID:                    Gemini31Pro,
		Name:                  "VertexAI: Gemini 3.1 Pro",
		Provider:              "vertexai",
		APIModel:              "gemini-3.1-pro",
		Currency:              "USD",
		CostPer1MIn:           2,
		CostPer1MOut:          12,
		CostPer1MInCached:     0.2,
		ContextWindow:         1048576,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}
