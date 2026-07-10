package model

// Groq provider identifier and hosted model IDs for this registry.
const (
	ProviderGROQ Provider = "groq"

	Llama4Scout           ID = "meta-llama/llama-4-scout-17b-16e-instruct"
	Llama3_3_70BVersatile ID = "llama-3.3-70b-versatile"
	Llama3_1_8BInstant    ID = "llama-3.1-8b-instant"
	GPTOss120B            ID = "openai/gpt-oss-120b"
	GPTOss20B             ID = "openai/gpt-oss-20b"
	Qwen3_32BGroq         ID = "qwen/qwen3-32b"
	KimiK2                ID = "moonshotai/kimi-k2-instruct-0905"
)

// GroqModels maps Groq model IDs to their configurations.
var GroqModels = map[ID]Model{
	Llama4Scout: {
		ID:                    Llama4Scout,
		Name:                  "Llama4Scout",
		Provider:              ProviderGROQ,
		APIModel:              "meta-llama/llama-4-scout-17b-16e-instruct",
		CostPer1MIn:           0.11,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.34,
		ContextWindow:         128_000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},

	Llama3_3_70BVersatile: {
		ID:                    Llama3_3_70BVersatile,
		Name:                  "Llama3_3_70BVersatile",
		Provider:              ProviderGROQ,
		APIModel:              "llama-3.3-70b-versatile",
		CostPer1MIn:           0.59,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.79,
		ContextWindow:         128_000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},

	Llama3_1_8BInstant: {
		ID:                    Llama3_1_8BInstant,
		Name:                  "Llama3_1_8BInstant",
		Provider:              ProviderGROQ,
		APIModel:              "llama-3.1-8b-instant",
		CostPer1MIn:           0.05,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.08,
		ContextWindow:         131_072,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},

	GPTOss120B: {
		ID:                    GPTOss120B,
		Name:                  "GPT-OSS 120B",
		Provider:              ProviderGROQ,
		APIModel:              "openai/gpt-oss-120b",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.60,
		ContextWindow:         131_072,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	GPTOss20B: {
		ID:                    GPTOss20B,
		Name:                  "GPT-OSS 20B",
		Provider:              ProviderGROQ,
		APIModel:              "openai/gpt-oss-20b",
		CostPer1MIn:           0.075,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.30,
		ContextWindow:         131_072,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Qwen3_32BGroq: {
		ID:                    Qwen3_32BGroq,
		Name:                  "Qwen3 32B",
		Provider:              ProviderGROQ,
		APIModel:              "qwen/qwen3-32b",
		CostPer1MIn:           0.29,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.59,
		ContextWindow:         131_072,
		DefaultMaxTokens:      40960,
		CanReason:             false,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	KimiK2: {
		ID:                    KimiK2,
		Name:                  "Kimi K2",
		Provider:              ProviderGROQ,
		APIModel:              "moonshotai/kimi-k2-instruct-0905",
		CostPer1MIn:           1.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          3.00,
		ContextWindow:         262_144,
		DefaultMaxTokens:      16384,
		CanReason:             false,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
