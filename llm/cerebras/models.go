package cerebras

import (
	"github.com/joakimcarlsson/ai/llm"
)

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
	Llama31_8B string = "cerebras.llama3.1-8b"
	GPTOss120B string = "cerebras.gpt-oss-120b"
)

// Models maps Cerebras model IDs to their configurations.
//
// Pricing source: https://inference-docs.cerebras.ai/models/ (per-model pages).
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Llama31_8B: {
		ID:                    Llama31_8B,
		Name:                  "Cerebras – Llama 3.1 8B",
		Provider:              "cerebras",
		APIModel:              "llama3.1-8b",
		CostPer1MIn:           0.1,
		CostPer1MOut:          0.1,
		ContextWindow:         32768,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	GPTOss120B: {
		ID:                    GPTOss120B,
		Name:                  "Cerebras – GPT-OSS 120B",
		Provider:              "cerebras",
		APIModel:              "gpt-oss-120b",
		CostPer1MIn:           0.35,
		CostPer1MOut:          0.75,
		ContextWindow:         131072,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
}
