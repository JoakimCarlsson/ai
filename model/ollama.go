package model

// Ollama provider identifier and a representative subset of locally-pulled
// open-weights model IDs for this registry.
//
// Ollama runs models locally; there is no API spend per request. The cost
// fields are zero across the board — real cost is electricity / GPU rental,
// not per-token API fees. Ollama supports any model the user has pulled
// locally; callers can pass any model id via [llmopenai.WithModel] even
// without a registered entry here.
const (
	ProviderOllama Provider = "ollama"

	OllamaLlama32_3B     ID = "ollama.llama3.2:3b"
	OllamaLlama33_70B    ID = "ollama.llama3.3:70b"
	OllamaQwen25_7B      ID = "ollama.qwen2.5:7b"
	OllamaQwen25_72B     ID = "ollama.qwen2.5:72b"
	OllamaDeepSeekR1_8B  ID = "ollama.deepseek-r1:8b"
	OllamaDeepSeekR1_70B ID = "ollama.deepseek-r1:70b"
	OllamaMistral7B      ID = "ollama.mistral:7b"
)

// OllamaModels maps Ollama model IDs to their configurations.
//
// Local inference; all per-token costs are zero. See https://ollama.com/library
// for the full library. Fetched: 2026-05-04.
var OllamaModels = map[ID]Model{
	OllamaLlama32_3B: {
		ID:                    OllamaLlama32_3B,
		Name:                  "Ollama – Llama 3.2 3B",
		Provider:              ProviderOllama,
		APIModel:              "llama3.2:3b",
		CostPer1MIn:           0,
		CostPer1MInCached:     0,
		CostPer1MOut:          0,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4_096,
		SupportsStructuredOut: true,
	},
	OllamaLlama33_70B: {
		ID:                    OllamaLlama33_70B,
		Name:                  "Ollama – Llama 3.3 70B",
		Provider:              ProviderOllama,
		APIModel:              "llama3.3:70b",
		CostPer1MIn:           0,
		CostPer1MInCached:     0,
		CostPer1MOut:          0,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4_096,
		SupportsStructuredOut: true,
	},
	OllamaQwen25_7B: {
		ID:                    OllamaQwen25_7B,
		Name:                  "Ollama – Qwen 2.5 7B",
		Provider:              ProviderOllama,
		APIModel:              "qwen2.5:7b",
		CostPer1MIn:           0,
		CostPer1MInCached:     0,
		CostPer1MOut:          0,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4_096,
		SupportsStructuredOut: true,
	},
	OllamaQwen25_72B: {
		ID:                    OllamaQwen25_72B,
		Name:                  "Ollama – Qwen 2.5 72B",
		Provider:              ProviderOllama,
		APIModel:              "qwen2.5:72b",
		CostPer1MIn:           0,
		CostPer1MInCached:     0,
		CostPer1MOut:          0,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4_096,
		SupportsStructuredOut: true,
	},
	OllamaDeepSeekR1_8B: {
		ID:                    OllamaDeepSeekR1_8B,
		Name:                  "Ollama – DeepSeek R1 Distill 8B",
		Provider:              ProviderOllama,
		APIModel:              "deepseek-r1:8b",
		CostPer1MIn:           0,
		CostPer1MInCached:     0,
		CostPer1MOut:          0,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	OllamaDeepSeekR1_70B: {
		ID:                    OllamaDeepSeekR1_70B,
		Name:                  "Ollama – DeepSeek R1 Distill 70B",
		Provider:              ProviderOllama,
		APIModel:              "deepseek-r1:70b",
		CostPer1MIn:           0,
		CostPer1MInCached:     0,
		CostPer1MOut:          0,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	OllamaMistral7B: {
		ID:                    OllamaMistral7B,
		Name:                  "Ollama – Mistral 7B",
		Provider:              ProviderOllama,
		APIModel:              "mistral:7b",
		CostPer1MIn:           0,
		CostPer1MInCached:     0,
		CostPer1MOut:          0,
		CostPer1MOutCached:    0,
		ContextWindow:         32_768,
		DefaultMaxTokens:      4_096,
		SupportsStructuredOut: false,
	},
}
