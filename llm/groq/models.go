package groq

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Groq provider identifier and hosted model IDs for this registry.
const (
	Llama4Scout           string = "meta-llama/llama-4-scout-17b-16e-instruct"
	Llama3_3_70BVersatile string = "llama-3.3-70b-versatile"
	Llama3_1_8BInstant    string = "llama-3.1-8b-instant"
	GPTOss120B            string = "openai/gpt-oss-120b"
	GPTOss20B             string = "openai/gpt-oss-20b"
	GPTOssSafeguard20B    string = "openai/gpt-oss-safeguard-20b"
	Qwen36_27BGroq        string = "qwen/qwen3.6-27b"
	KimiK2                string = "moonshotai/kimi-k2-instruct-0905"
)

// Models maps Groq model IDs to their configurations.
//
// Pricing source: https://console.groq.com/docs/models and
// https://groq.com/pricing.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Llama4Scout: {
		ID:                    Llama4Scout,
		Name:                  "Llama4Scout",
		Provider:              "groq",
		APIModel:              "meta-llama/llama-4-scout-17b-16e-instruct",
		Currency:              "USD",
		CostPer1MIn:           0.11,
		CostPer1MOut:          0.34,
		ContextWindow:         128000,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Llama3_3_70BVersatile: {
		ID:            Llama3_3_70BVersatile,
		Name:          "Llama3_3_70BVersatile",
		Provider:      "groq",
		APIModel:      "llama-3.3-70b-versatile",
		Currency:      "USD",
		CostPer1MIn:   0.59,
		CostPer1MOut:  0.79,
		ContextWindow: 128000,
	},
	Llama3_1_8BInstant: {
		ID:            Llama3_1_8BInstant,
		Name:          "Llama3_1_8BInstant",
		Provider:      "groq",
		APIModel:      "llama-3.1-8b-instant",
		Currency:      "USD",
		CostPer1MIn:   0.05,
		CostPer1MOut:  0.08,
		ContextWindow: 131072,
	},
	GPTOss120B: {
		ID:               GPTOss120B,
		Name:             "GPT-OSS 120B",
		Provider:         "groq",
		APIModel:         "openai/gpt-oss-120b",
		Currency:         "USD",
		CostPer1MIn:      0.15,
		CostPer1MOut:     0.6,
		ContextWindow:    131072,
		DefaultMaxTokens: 65536,
		CanReason:        true,
	},
	GPTOss20B: {
		ID:               GPTOss20B,
		Name:             "GPT-OSS 20B",
		Provider:         "groq",
		APIModel:         "openai/gpt-oss-20b",
		Currency:         "USD",
		CostPer1MIn:      0.075,
		CostPer1MOut:     0.3,
		ContextWindow:    131072,
		DefaultMaxTokens: 65536,
		CanReason:        true,
	},
	GPTOssSafeguard20B: {
		ID:                    GPTOssSafeguard20B,
		Name:                  "GPT-OSS Safeguard 20B",
		Provider:              "groq",
		APIModel:              "openai/gpt-oss-safeguard-20b",
		Currency:              "USD",
		CostPer1MIn:           0.075,
		CostPer1MOut:          0.3,
		ContextWindow:         131072,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen36_27BGroq: {
		ID:                    Qwen36_27BGroq,
		Name:                  "Qwen3.6 27B",
		Provider:              "groq",
		APIModel:              "qwen/qwen3.6-27b",
		Currency:              "USD",
		CostPer1MIn:           0.6,
		CostPer1MOut:          3,
		ContextWindow:         131072,
		DefaultMaxTokens:      16384,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	KimiK2: {
		ID:               KimiK2,
		Name:             "Kimi K2",
		Provider:         "groq",
		APIModel:         "moonshotai/kimi-k2-instruct-0905",
		Currency:         "USD",
		CostPer1MIn:      1,
		CostPer1MOut:     3,
		ContextWindow:    262144,
		DefaultMaxTokens: 16384,
	},
}
