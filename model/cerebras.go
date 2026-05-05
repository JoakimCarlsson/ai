package model

// Cerebras provider identifier and currently-listed chat model IDs for this
// registry. Cerebras' marketing pricing page is tier-only (Free / Developer /
// Enterprise); the per-token rates published below come from the per-model
// pages under https://inference-docs.cerebras.ai/models/.
//
// Cerebras' supported-models list rotates relatively often; preview-tier and
// older entries (e.g. Llama 3.3 70B, Llama 3.1 405B) have been dropped from
// the active list and are intentionally not catalogued here. Z.ai GLM 4.7 is
// listed by Cerebras but its per-token price is not published on cerebras.ai
// itself, so it is omitted to avoid recording an unverified figure.
const (
	ProviderCerebras Provider = "cerebras"

	CerebrasLlama31_8B ID = "cerebras.llama3.1-8b"
	CerebrasGPTOss120B ID = "cerebras.gpt-oss-120b"
	CerebrasQwen3_235B ID = "cerebras.qwen-3-235b-a22b-instruct-2507"
)

// CerebrasModels maps Cerebras model IDs to their configurations.
//
// Pricing source: https://inference-docs.cerebras.ai/models/ (per-model pages).
// Fetched: 2026-05-04.
var CerebrasModels = map[ID]Model{
	CerebrasLlama31_8B: {
		ID:                    CerebrasLlama31_8B,
		Name:                  "Cerebras – Llama 3.1 8B",
		Provider:              ProviderCerebras,
		APIModel:              "llama3.1-8b",
		CostPer1MIn:           0.10,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.10,
		CostPer1MOutCached:    0,
		ContextWindow:         32_768,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	CerebrasGPTOss120B: {
		ID:                    CerebrasGPTOss120B,
		Name:                  "Cerebras – GPT-OSS 120B",
		Provider:              ProviderCerebras,
		APIModel:              "gpt-oss-120b",
		CostPer1MIn:           0.35,
		CostPer1MInCached:     0,
		CostPer1MOut:          0.75,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      32_768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	CerebrasQwen3_235B: {
		ID:                    CerebrasQwen3_235B,
		Name:                  "Cerebras – Qwen 3 235B Instruct",
		Provider:              ProviderCerebras,
		APIModel:              "qwen-3-235b-a22b-instruct-2507",
		CostPer1MIn:           0.60,
		CostPer1MInCached:     0,
		CostPer1MOut:          1.20,
		CostPer1MOutCached:    0,
		ContextWindow:         131_072,
		DefaultMaxTokens:      32_768,
		SupportsStructuredOut: true,
	},
}
