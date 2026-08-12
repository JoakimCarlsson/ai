package berget

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Berget AI (https://berget.ai) is a Swedish, EU-hosted inference provider with
// an OpenAI-compatible API at https://api.berget.ai/v1.
//
// All prices below are in EUR, not USD: Berget bills in EUR and the Cost*
// fields hold the raw EUR figures from the /v1/models API (fetched 2026-06-30).
// The API does not return context windows, so ContextWindow values come from
// the upstream model cards (131_072 where a model's window is unpublished).
const (
	GPTOSS120B      string = "openai/gpt-oss-120b"
	MistralMedium35 string = "mistralai/Mistral-Medium-3.5-128B"
	MistralSmall32  string = "mistralai/Mistral-Small-3.2-24B-Instruct-2506"
	GLM47           string = "zai-org/GLM-4.7-FP8"
	GLM52           string = "zai-org/GLM-5.2"
	KimiK26         string = "moonshotai/Kimi-K2.6"
	Gemma431B       string = "google/gemma-4-31B-it"
	Llama3370B      string = "meta-llama/Llama-3.3-70B-Instruct"
)

// Models maps Berget chat model IDs to their configurations.
// Prices are EUR per 1M tokens.
//
// Pricing source: https://api.berget.ai/v1/models (model IDs; Berget does
// not publish per-model rates).
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	GPTOSS120B: {
		ID:                    GPTOSS120B,
		Name:                  "GPT-OSS 120B",
		Provider:              "berget",
		APIModel:              "openai/gpt-oss-120b",
		CostPer1MIn:           0.2,
		CostPer1MOut:          0.75,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	MistralMedium35: {
		ID:                    MistralMedium35,
		Name:                  "Mistral Medium 3.5",
		Provider:              "berget",
		APIModel:              "mistralai/Mistral-Medium-3.5-128B",
		CostPer1MIn:           1.5,
		CostPer1MOut:          5,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	MistralSmall32: {
		ID:                    MistralSmall32,
		Name:                  "Mistral Small 3.2 24B",
		Provider:              "berget",
		APIModel:              "mistralai/Mistral-Small-3.2-24B-Instruct-2506",
		CostPer1MIn:           0.3,
		CostPer1MOut:          0.3,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	GLM47: {
		ID:                    GLM47,
		Name:                  "GLM-4.7 FP8",
		Provider:              "berget",
		APIModel:              "zai-org/GLM-4.7-FP8",
		CostPer1MIn:           0.7,
		CostPer1MOut:          2.5,
		ContextWindow:         200000,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	GLM52: {
		ID:                    GLM52,
		Name:                  "GLM-5.2",
		Provider:              "berget",
		APIModel:              "zai-org/GLM-5.2",
		CostPer1MIn:           1.4,
		CostPer1MOut:          4.4,
		ContextWindow:         200000,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	KimiK26: {
		ID:                    KimiK26,
		Name:                  "Kimi K2.6",
		Provider:              "berget",
		APIModel:              "moonshotai/Kimi-K2.6",
		CostPer1MIn:           0.75,
		CostPer1MOut:          3.5,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemma431B: {
		ID:                    Gemma431B,
		Name:                  "Gemma 4 31B",
		Provider:              "berget",
		APIModel:              "google/gemma-4-31B-it",
		CostPer1MIn:           0.25,
		CostPer1MOut:          0.5,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Llama3370B: {
		ID:                    Llama3370B,
		Name:                  "Llama 3.3 70B Instruct",
		Provider:              "berget",
		APIModel:              "meta-llama/Llama-3.3-70B-Instruct",
		CostPer1MIn:           0.9,
		CostPer1MOut:          0.9,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
}
