package model

// Fireworks provider identifier and a curated set of hosted chat model IDs
// for this registry.
//
// Fireworks hosts far more models than are catalogued here; callers can pass
// any "accounts/fireworks/models/..." path via [llmopenai.WithModel] even
// without a registered entry. Reasoning-class entries (DeepSeek R1, Qwen3
// thinking variants, GPT-OSS) carry Fireworks' "Logic / Long Context"
// surcharge with split input/output rates.
const (
	ProviderFireworks Provider = "fireworks"

	FireworksLlama31_70B          ID = "fireworks.llama-v3p1-70b-instruct"
	FireworksLlama33_70B          ID = "fireworks.llama-v3p3-70b-instruct"
	FireworksDeepSeekV3           ID = "fireworks.deepseek-v3"
	FireworksDeepSeekV3p1Terminus ID = "fireworks.deepseek-v3p1-terminus"
	FireworksDeepSeekR1           ID = "fireworks.deepseek-r1"
	FireworksQwen25_72B           ID = "fireworks.qwen2p5-72b-instruct"
	FireworksQwen3_235BInstruct   ID = "fireworks.qwen3-235b-a22b-instruct-2507"
	FireworksQwen3_30BThinking    ID = "fireworks.qwen3-30b-a3b-thinking-2507"
	FireworksMixtral8x22B         ID = "fireworks.mixtral-8x22b-instruct"
	FireworksKimiK2               ID = "fireworks.kimi-k2-instruct"
	FireworksGPTOss120B           ID = "fireworks.gpt-oss-120b"
	FireworksGPTOss20B            ID = "fireworks.gpt-oss-20b"
)

// FireworksModels maps Fireworks model IDs to their configurations.
//
// Pricing source: https://fireworks.ai/pricing. Fetched: 2026-05-04.
var FireworksModels = map[ID]Model{
	FireworksLlama31_70B: {
		ID:                    FireworksLlama31_70B,
		Name:                  "Fireworks – Llama 3.1 70B Instruct",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/llama-v3p1-70b-instruct",
		CostPer1MIn:           0.90,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.90,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	FireworksLlama33_70B: {
		ID:                    FireworksLlama33_70B,
		Name:                  "Fireworks – Llama 3.3 70B Instruct",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/llama-v3p3-70b-instruct",
		CostPer1MIn:           0.90,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.90,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	FireworksDeepSeekV3: {
		ID:                    FireworksDeepSeekV3,
		Name:                  "Fireworks – DeepSeek V3",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/deepseek-v3",
		CostPer1MIn:           0.56,
		CostPer1MInCached:     0,
		CostPer1MOut:          1.68,
		CostPer1MOutCached:    0,
		ContextWindow:         163_840,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	FireworksDeepSeekV3p1Terminus: {
		ID:                    FireworksDeepSeekV3p1Terminus,
		Name:                  "Fireworks – DeepSeek V3.1 Terminus",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/deepseek-v3p1-terminus",
		CostPer1MIn:           0.56,
		CostPer1MInCached:     0,
		CostPer1MOut:          1.68,
		CostPer1MOutCached:    0,
		ContextWindow:         163_840,
		DefaultMaxTokens:      8192,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	FireworksDeepSeekR1: {
		ID:                    FireworksDeepSeekR1,
		Name:                  "Fireworks – DeepSeek R1",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/deepseek-r1",
		CostPer1MIn:           3.00,
		CostPer1MInCached:     0,
		CostPer1MOut:          8.00,
		CostPer1MOutCached:    0,
		ContextWindow:         163_840,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	FireworksQwen25_72B: {
		ID:                    FireworksQwen25_72B,
		Name:                  "Fireworks – Qwen 2.5 72B Instruct",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/qwen2p5-72b-instruct",
		CostPer1MIn:           0.90,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.90,
		CostPer1MOutCached:    0,
		ContextWindow:         32_768,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	FireworksQwen3_235BInstruct: {
		ID:                    FireworksQwen3_235BInstruct,
		Name:                  "Fireworks – Qwen 3 235B Instruct (2507)",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/qwen3-235b-a22b-instruct-2507",
		CostPer1MIn:           0.22,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.88,
		CostPer1MOutCached:    0,
		ContextWindow:         262_144,
		DefaultMaxTokens:      32_768,
		SupportsStructuredOut: true,
	},
	FireworksQwen3_30BThinking: {
		ID:                    FireworksQwen3_30BThinking,
		Name:                  "Fireworks – Qwen 3 30B A3B Thinking (2507)",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/qwen3-30b-a3b-thinking-2507",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.60,
		CostPer1MOutCached:    0,
		ContextWindow:         262_144,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	FireworksMixtral8x22B: {
		ID:                    FireworksMixtral8x22B,
		Name:                  "Fireworks – Mixtral 8x22B Instruct",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/mixtral-8x22b-instruct",
		CostPer1MIn:           1.20,
		CostPer1MInCached:     0,
		CostPer1MOut:          1.20,
		CostPer1MOutCached:    0,
		ContextWindow:         65_536,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	FireworksKimiK2: {
		ID:                    FireworksKimiK2,
		Name:                  "Fireworks – Kimi K2 Instruct",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/kimi-k2-instruct",
		CostPer1MIn:           0.60,
		CostPer1MInCached:     0,
		CostPer1MOut:          3.00,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      16_384,
		SupportsStructuredOut: true,
	},
	FireworksGPTOss120B: {
		ID:                    FireworksGPTOss120B,
		Name:                  "Fireworks – GPT-OSS 120B",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/gpt-oss-120b",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.60,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      65_536,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	FireworksGPTOss20B: {
		ID:                    FireworksGPTOss20B,
		Name:                  "Fireworks – GPT-OSS 20B",
		Provider:              ProviderFireworks,
		APIModel:              "accounts/fireworks/models/gpt-oss-20b",
		CostPer1MIn:           0.07,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.30,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      65_536,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
}
