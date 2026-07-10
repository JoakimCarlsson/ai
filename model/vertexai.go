package model

// Vertex AI provider identifier and Gemini model IDs for this registry.
const (
	ProviderVertexAI Provider = "vertexai"

	VertexAIGemini25Flash ID = "vertexai.gemini-2.5-flash"
	VertexAIGemini25      ID = "vertexai.gemini-2.5"

	VertexAIGemini35Flash     ID = "vertexai.gemini-3.5-flash"
	VertexAIGemini31FlashLite ID = "vertexai.gemini-3.1-flash-lite"
	VertexAIGemini3Pro        ID = "vertexai.gemini-3-pro"
	VertexAIGemini25FlashLite ID = "vertexai.gemini-2.5-flash-lite"
	VertexAIGemini20Flash     ID = "vertexai.gemini-2.0-flash"
	VertexAIGemini20FlashLite ID = "vertexai.gemini-2.0-flash-lite"
)

// VertexAIGeminiModels maps Vertex AI Gemini model IDs to their configurations.
var VertexAIGeminiModels = map[ID]Model{
	VertexAIGemini25Flash: {
		ID:                    VertexAIGemini25Flash,
		Name:                  "VertexAI: Gemini 2.5 Flash",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-2.5-flash-preview-04-17",
		CostPer1MIn:           GeminiModels[Gemini25Flash].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini25Flash].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini25Flash].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini25Flash].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini25Flash].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini25Flash].DefaultMaxTokens,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	VertexAIGemini25: {
		ID:                    VertexAIGemini25,
		Name:                  "VertexAI: Gemini 2.5 Pro",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-2.5-pro-preview-03-25",
		CostPer1MIn:           GeminiModels[Gemini25].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini25].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini25].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini25].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini25].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini25].DefaultMaxTokens,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	VertexAIGemini35Flash: {
		ID:                    VertexAIGemini35Flash,
		Name:                  "VertexAI: Gemini 3.5 Flash",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-3.5-flash",
		CostPer1MIn:           GeminiModels[Gemini35Flash].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini35Flash].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini35Flash].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini35Flash].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini35Flash].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini35Flash].DefaultMaxTokens,
		CanReason:             GeminiModels[Gemini35Flash].CanReason,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	VertexAIGemini31FlashLite: {
		ID:                    VertexAIGemini31FlashLite,
		Name:                  "VertexAI: Gemini 3.1 Flash Lite",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-3.1-flash-lite",
		CostPer1MIn:           GeminiModels[Gemini31FlashLite].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini31FlashLite].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini31FlashLite].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini31FlashLite].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini31FlashLite].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini31FlashLite].DefaultMaxTokens,
		CanReason:             GeminiModels[Gemini31FlashLite].CanReason,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	VertexAIGemini3Pro: {
		ID:                    VertexAIGemini3Pro,
		Name:                  "VertexAI: Gemini 3 Pro",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-3-pro",
		CostPer1MIn:           GeminiModels[Gemini3Pro].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini3Pro].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini3Pro].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini3Pro].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini3Pro].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini3Pro].DefaultMaxTokens,
		CanReason:             GeminiModels[Gemini3Pro].CanReason,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	VertexAIGemini25FlashLite: {
		ID:                    VertexAIGemini25FlashLite,
		Name:                  "VertexAI: Gemini 2.5 Flash Lite",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-2.5-flash-lite",
		CostPer1MIn:           GeminiModels[Gemini25FlashLite].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini25FlashLite].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini25FlashLite].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini25FlashLite].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini25FlashLite].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini25FlashLite].DefaultMaxTokens,
		CanReason:             GeminiModels[Gemini25FlashLite].CanReason,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	VertexAIGemini20Flash: {
		ID:                    VertexAIGemini20Flash,
		Name:                  "VertexAI: Gemini 2.0 Flash",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-2.0-flash",
		CostPer1MIn:           GeminiModels[Gemini20Flash].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini20Flash].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini20Flash].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini20Flash].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini20Flash].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini20Flash].DefaultMaxTokens,
		CanReason:             GeminiModels[Gemini20Flash].CanReason,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	VertexAIGemini20FlashLite: {
		ID:                    VertexAIGemini20FlashLite,
		Name:                  "VertexAI: Gemini 2.0 Flash Lite",
		Provider:              ProviderVertexAI,
		APIModel:              "gemini-2.0-flash-lite",
		CostPer1MIn:           GeminiModels[Gemini20FlashLite].CostPer1MIn,
		CostPer1MInCached:     GeminiModels[Gemini20FlashLite].CostPer1MInCached,
		CostPer1MOut:          GeminiModels[Gemini20FlashLite].CostPer1MOut,
		CostPer1MOutCached:    GeminiModels[Gemini20FlashLite].CostPer1MOutCached,
		ContextWindow:         GeminiModels[Gemini20FlashLite].ContextWindow,
		DefaultMaxTokens:      GeminiModels[Gemini20FlashLite].DefaultMaxTokens,
		CanReason:             GeminiModels[Gemini20FlashLite].CanReason,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}
