package model

// Cerebras provider identifier and currently-listed chat model IDs for this
// registry. Cerebras' marketing pricing page is tier-only (Free / Developer /
// Enterprise); the per-token rates published below come from the per-model
// pages under https://inference-docs.cerebras.ai/models/.
//
// Cerebras' supported-models list rotates relatively often; preview-tier and
// older entries (e.g. Llama 3.3 70B, Llama 3.1 405B, Qwen 3 235B Instruct)
// have been dropped from the active list and are intentionally not catalogued
// here. Gemma 4 31B and Z.ai GLM 4.7 are listed by Cerebras but their
// per-token prices are not published on cerebras.ai itself, so they are
// omitted to avoid recording unverified figures.
const (
	ProviderCerebras Provider = "cerebras"

	CerebrasLlama31_8B ID = "cerebras.llama3.1-8b"
	CerebrasGPTOss120B ID = "cerebras.gpt-oss-120b"
)

// CerebrasModels maps Cerebras model IDs to their configurations.
//
// Pricing source: https://inference-docs.cerebras.ai/models/ (per-model pages).
// Fetched: 2026-07-26.
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
}
