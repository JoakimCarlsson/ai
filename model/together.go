package model

// Together provider identifier and a curated set of hosted chat model IDs
// for this registry.
//
// Together hosts far more models than are catalogued here; callers can pass
// any hosted model id via [llmopenai.WithModel] even without a registered
// entry. Mixtral 8x7B has been removed from Together's serverless pricing
// list and is intentionally not catalogued here.
const (
	ProviderTogether Provider = "together"

	TogetherLlama33_70B    ID = "together.meta-llama/Llama-3.3-70B-Instruct-Turbo"
	TogetherLlama3_8BLite  ID = "together.meta-llama/Meta-Llama-3-8B-Instruct-Lite"
	TogetherDeepSeekV31    ID = "together.deepseek-ai/DeepSeek-V3.1"
	TogetherDeepSeekV4Pro  ID = "together.deepseek-ai/DeepSeek-V4-Pro"
	TogetherDeepSeekR1     ID = "together.deepseek-ai/DeepSeek-R1"
	TogetherQwen35_397B    ID = "together.Qwen/Qwen3.5-397B-A17B"
	TogetherQwen3Coder480B ID = "together.Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8"
	TogetherQwen25_7BTurbo ID = "together.Qwen/Qwen2.5-7B-Instruct-Turbo"
	TogetherKimiK2_6       ID = "together.moonshotai/Kimi-K2.6"
	TogetherKimiK2_5       ID = "together.moonshotai/Kimi-K2.5"
	TogetherGPTOss120B     ID = "together.openai/gpt-oss-120b"
	TogetherGLM5_1         ID = "together.zai-org/GLM-5.1"
)

// TogetherModels maps Together AI model IDs to their configurations.
//
// Pricing source: https://www.together.ai/pricing. Fetched: 2026-05-04.
var TogetherModels = map[ID]Model{
	TogetherLlama33_70B: {
		ID:                    TogetherLlama33_70B,
		Name:                  "Together – Llama 3.3 70B Instruct Turbo",
		Provider:              ProviderTogether,
		APIModel:              "meta-llama/Llama-3.3-70B-Instruct-Turbo",
		CostPer1MIn:           1.04,
		CostPer1MInCached:     0,
		CostPer1MOut:          1.04,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	TogetherLlama3_8BLite: {
		ID:                    TogetherLlama3_8BLite,
		Name:                  "Together – Llama 3 8B Instruct Lite",
		Provider:              ProviderTogether,
		APIModel:              "meta-llama/Meta-Llama-3-8B-Instruct-Lite",
		CostPer1MIn:           0.14,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.14,
		CostPer1MOutCached:    0,
		ContextWindow:         8_192,
		DefaultMaxTokens:      4_096,
		SupportsStructuredOut: true,
	},
	TogetherDeepSeekV31: {
		ID:                    TogetherDeepSeekV31,
		Name:                  "Together – DeepSeek V3.1",
		Provider:              ProviderTogether,
		APIModel:              "deepseek-ai/DeepSeek-V3.1",
		CostPer1MIn:           0.60,
		CostPer1MInCached:     0,
		CostPer1MOut:          1.70,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	TogetherDeepSeekV4Pro: {
		ID:                    TogetherDeepSeekV4Pro,
		Name:                  "Together – DeepSeek V4 Pro",
		Provider:              ProviderTogether,
		APIModel:              "deepseek-ai/DeepSeek-V4-Pro",
		CostPer1MIn:           1.74,
		CostPer1MInCached:     0,
		CostPer1MOut:          3.48,
		CostPer1MOutCached:    0,
		ContextWindow:         512_000,
		DefaultMaxTokens:      32_768,
		SupportsStructuredOut: true,
	},
	TogetherDeepSeekR1: {
		ID:                    TogetherDeepSeekR1,
		Name:                  "Together – DeepSeek R1",
		Provider:              ProviderTogether,
		APIModel:              "deepseek-ai/DeepSeek-R1",
		CostPer1MIn:           3.00,
		CostPer1MInCached:     0,
		CostPer1MOut:          7.00,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	TogetherQwen35_397B: {
		ID:                    TogetherQwen35_397B,
		Name:                  "Together – Qwen 3.5 397B A17B",
		Provider:              ProviderTogether,
		APIModel:              "Qwen/Qwen3.5-397B-A17B",
		CostPer1MIn:           0.60,
		CostPer1MInCached:     0,
		CostPer1MOut:          3.60,
		CostPer1MOutCached:    0,
		ContextWindow:         262_144,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	TogetherQwen3Coder480B: {
		ID:                    TogetherQwen3Coder480B,
		Name:                  "Together – Qwen 3 Coder 480B",
		Provider:              ProviderTogether,
		APIModel:              "Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0,
		CostPer1MOut:          2.00,
		CostPer1MOutCached:    0,
		ContextWindow:         256_000,
		DefaultMaxTokens:      32_768,
		SupportsStructuredOut: true,
	},
	TogetherQwen25_7BTurbo: {
		ID:                    TogetherQwen25_7BTurbo,
		Name:                  "Together – Qwen 2.5 7B Instruct Turbo",
		Provider:              ProviderTogether,
		APIModel:              "Qwen/Qwen2.5-7B-Instruct-Turbo",
		CostPer1MIn:           0.30,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.30,
		CostPer1MOutCached:    0,
		ContextWindow:         32_768,
		DefaultMaxTokens:      4_096,
		SupportsStructuredOut: true,
	},
	TogetherKimiK2_6: {
		ID:                    TogetherKimiK2_6,
		Name:                  "Together – Kimi K2.6",
		Provider:              ProviderTogether,
		APIModel:              "moonshotai/Kimi-K2.6",
		CostPer1MIn:           1.20,
		CostPer1MInCached:     0,
		CostPer1MOut:          4.50,
		CostPer1MOutCached:    0,
		ContextWindow:         262_144,
		DefaultMaxTokens:      16_384,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	TogetherKimiK2_5: {
		ID:                    TogetherKimiK2_5,
		Name:                  "Together – Kimi K2.5",
		Provider:              ProviderTogether,
		APIModel:              "moonshotai/Kimi-K2.5",
		CostPer1MIn:           0.50,
		CostPer1MInCached:     0,
		CostPer1MOut:          2.80,
		CostPer1MOutCached:    0,
		ContextWindow:         262_144,
		DefaultMaxTokens:      16_384,
		SupportsStructuredOut: true,
	},
	TogetherGPTOss120B: {
		ID:                    TogetherGPTOss120B,
		Name:                  "Together – GPT-OSS 120B",
		Provider:              ProviderTogether,
		APIModel:              "openai/gpt-oss-120b",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.60,
		CostPer1MOutCached:    0,
		ContextWindow:         128_000,
		DefaultMaxTokens:      65_536,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	TogetherGLM5_1: {
		ID:                    TogetherGLM5_1,
		Name:                  "Together – GLM 5.1",
		Provider:              ProviderTogether,
		APIModel:              "zai-org/GLM-5.1",
		CostPer1MIn:           1.40,
		CostPer1MInCached:     0,
		CostPer1MOut:          4.40,
		CostPer1MOutCached:    0,
		ContextWindow:         202_752,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
}
