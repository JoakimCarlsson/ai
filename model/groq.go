package model

const (
	ProviderGROQ ModelProvider = "groq"

	QWENQwq ModelID = "qwen-qwq"

	Llama4Scout               ModelID = "meta-llama/llama-4-scout-17b-16e-instruct"
	Llama4Maverick            ModelID = "meta-llama/llama-4-maverick-17b-128e-instruct"
	Llama3_3_70BVersatile     ModelID = "llama-3.3-70b-versatile"
	DeepseekR1DistillLlama70b ModelID = "deepseek-r1-distill-llama-70b"
	GPTOss120B                ModelID = "openai/gpt-oss-120b"
	GPTOss20B                 ModelID = "openai/gpt-oss-20b"
	Qwen3_32BGroq             ModelID = "qwen/qwen3-32b"
	KimiK2                    ModelID = "moonshotai/kimi-k2-instruct-0905"
)

var GroqModels = map[ModelID]Model{
	QWENQwq: {
		ID:                    QWENQwq,
		Name:                  "Qwen Qwq",
		Provider:              ProviderGROQ,
		APIModel:              "qwen-qwq-32b",
		CostPer1MIn:           0.29,
		CostPer1MInCached:     0.275,
		CostPer1MOutCached:    0.0,
		CostPer1MOut:          0.39,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		CanReason:             false,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},

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

	Llama4Maverick: {
		ID:                    Llama4Maverick,
		Name:                  "Llama4Maverick",
		Provider:              ProviderGROQ,
		APIModel:              "meta-llama/llama-4-maverick-17b-128e-instruct",
		CostPer1MIn:           0.20,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.20,
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

	DeepseekR1DistillLlama70b: {
		ID:                    DeepseekR1DistillLlama70b,
		Name:                  "DeepseekR1DistillLlama70b",
		Provider:              ProviderGROQ,
		APIModel:              "deepseek-r1-distill-llama-70b",
		CostPer1MIn:           0.75,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.99,
		ContextWindow:         128_000,
		CanReason:             true,
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
