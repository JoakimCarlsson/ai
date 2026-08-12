package together

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Together provider identifier and a curated set of hosted chat model IDs
// for this registry.
//
// Together hosts far more models than are catalogued here; callers can pass
// any hosted model id via [llmopenai.WithModel] even without a registered
// entry. Mixtral 8x7B has been removed from Together's serverless pricing
// list and is intentionally not catalogued here.
const (
	Llama33_70B    string = "together.meta-llama/Llama-3.3-70B-Instruct-Turbo"
	Llama3_8BLite  string = "together.meta-llama/Meta-Llama-3-8B-Instruct-Lite"
	DeepSeekV31    string = "together.deepseek-ai/DeepSeek-V3.1"
	DeepSeekV4Pro  string = "together.deepseek-ai/DeepSeek-V4-Pro"
	DeepSeekR1     string = "together.deepseek-ai/DeepSeek-R1"
	Qwen37Max      string = "together.Qwen/Qwen3.7-Max"
	Qwen37Plus     string = "together.Qwen/Qwen3.7-Plus"
	Qwen35_9B      string = "together.Qwen/Qwen3.5-9B"
	MiniMaxM3      string = "together.MiniMaxAI/MiniMax-M3"
	Qwen35_397B    string = "together.Qwen/Qwen3.5-397B-A17B"
	Qwen3Coder480B string = "together.Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8"
	Qwen25_7BTurbo string = "together.Qwen/Qwen2.5-7B-Instruct-Turbo"
	KimiK2_7Code   string = "together.moonshotai/Kimi-K2.7-Code"
	KimiK2_6       string = "together.moonshotai/Kimi-K2.6"
	KimiK2_5       string = "together.moonshotai/Kimi-K2.5"
	GPTOss120B     string = "together.openai/gpt-oss-120b"
	GPTOss20B      string = "together.openai/gpt-oss-20b"
	GLM51          string = "together.zai-org/GLM-5.1"
	GLM52          string = "together.zai-org/GLM-5.2"
)

// Models maps Together AI model IDs to their configurations.
//
// Pricing source: https://www.together.ai/pricing. Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Llama33_70B: {
		ID:                    Llama33_70B,
		Name:                  "Together – Llama 3.3 70B Instruct Turbo",
		Provider:              "together",
		APIModel:              "meta-llama/Llama-3.3-70B-Instruct-Turbo",
		Currency:              "USD",
		CostPer1MIn:           1.04,
		CostPer1MOut:          1.04,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	Llama3_8BLite: {
		ID:                    Llama3_8BLite,
		Name:                  "Together – Llama 3 8B Instruct Lite",
		Provider:              "together",
		APIModel:              "meta-llama/Meta-Llama-3-8B-Instruct-Lite",
		Currency:              "USD",
		CostPer1MIn:           0.14,
		CostPer1MOut:          0.14,
		ContextWindow:         8192,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	DeepSeekV31: {
		ID:                    DeepSeekV31,
		Name:                  "Together – DeepSeek V3.1",
		Provider:              "together",
		APIModel:              "deepseek-ai/DeepSeek-V3.1",
		Currency:              "USD",
		CostPer1MIn:           0.6,
		CostPer1MOut:          1.7,
		ContextWindow:         128000,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	DeepSeekV4Pro: {
		ID:                    DeepSeekV4Pro,
		Name:                  "Together – DeepSeek V4 Pro",
		Provider:              "together",
		APIModel:              "deepseek-ai/DeepSeek-V4-Pro",
		Currency:              "USD",
		CostPer1MIn:           1.74,
		CostPer1MOut:          3.48,
		ContextWindow:         512000,
		DefaultMaxTokens:      32768,
		SupportsStructuredOut: true,
	},
	DeepSeekR1: {
		ID:                    DeepSeekR1,
		Name:                  "Together – DeepSeek R1",
		Provider:              "together",
		APIModel:              "deepseek-ai/DeepSeek-R1",
		Currency:              "USD",
		CostPer1MIn:           3,
		CostPer1MOut:          7,
		ContextWindow:         131072,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen37Max: {
		ID:                    Qwen37Max,
		Name:                  "Together – Qwen 3.7 Max",
		Provider:              "together",
		APIModel:              "Qwen/Qwen3.7-Max",
		Currency:              "USD",
		CostPer1MIn:           1.25,
		CostPer1MOut:          3.75,
		ContextWindow:         1048576,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen37Plus: {
		ID:                    Qwen37Plus,
		Name:                  "Together – Qwen 3.7 Plus",
		Provider:              "together",
		APIModel:              "Qwen/Qwen3.7-Plus",
		Currency:              "USD",
		CostPer1MIn:           0.32,
		CostPer1MOut:          1.28,
		ContextWindow:         1048576,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Qwen35_9B: {
		ID:                    Qwen35_9B,
		Name:                  "Together – Qwen 3.5 9B",
		Provider:              "together",
		APIModel:              "Qwen/Qwen3.5-9B",
		Currency:              "USD",
		CostPer1MIn:           0.17,
		CostPer1MOut:          0.25,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	MiniMaxM3: {
		ID:                    MiniMaxM3,
		Name:                  "Together – MiniMax M3",
		Provider:              "together",
		APIModel:              "MiniMaxAI/MiniMax-M3",
		Currency:              "USD",
		CostPer1MIn:           0.3,
		CostPer1MOut:          1.2,
		CostPer1MInCached:     0.06,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen35_397B: {
		ID:                    Qwen35_397B,
		Name:                  "Together – Qwen 3.5 397B A17B",
		Provider:              "together",
		APIModel:              "Qwen/Qwen3.5-397B-A17B",
		Currency:              "USD",
		CostPer1MIn:           0.6,
		CostPer1MOut:          3.6,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen3Coder480B: {
		ID:                    Qwen3Coder480B,
		Name:                  "Together – Qwen 3 Coder 480B",
		Provider:              "together",
		APIModel:              "Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8",
		Currency:              "USD",
		CostPer1MIn:           2,
		CostPer1MOut:          2,
		ContextWindow:         256000,
		DefaultMaxTokens:      32768,
		SupportsStructuredOut: true,
	},
	Qwen25_7BTurbo: {
		ID:                    Qwen25_7BTurbo,
		Name:                  "Together – Qwen 2.5 7B Instruct Turbo",
		Provider:              "together",
		APIModel:              "Qwen/Qwen2.5-7B-Instruct-Turbo",
		Currency:              "USD",
		CostPer1MIn:           0.3,
		CostPer1MOut:          0.3,
		ContextWindow:         32768,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	KimiK2_7Code: {
		ID:                    KimiK2_7Code,
		Name:                  "Together – Kimi K2.7 Code",
		Provider:              "together",
		APIModel:              "moonshotai/Kimi-K2.7-Code",
		Currency:              "USD",
		CostPer1MIn:           0.95,
		CostPer1MOut:          4,
		ContextWindow:         262144,
		DefaultMaxTokens:      16384,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	KimiK2_6: {
		ID:                    KimiK2_6,
		Name:                  "Together – Kimi K2.6",
		Provider:              "together",
		APIModel:              "moonshotai/Kimi-K2.6",
		Currency:              "USD",
		CostPer1MIn:           1.2,
		CostPer1MOut:          4.5,
		ContextWindow:         262144,
		DefaultMaxTokens:      16384,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	KimiK2_5: {
		ID:                    KimiK2_5,
		Name:                  "Together – Kimi K2.5",
		Provider:              "together",
		APIModel:              "moonshotai/Kimi-K2.5",
		Currency:              "USD",
		CostPer1MIn:           0.5,
		CostPer1MOut:          2.8,
		ContextWindow:         262144,
		DefaultMaxTokens:      16384,
		SupportsStructuredOut: true,
	},
	GPTOss120B: {
		ID:                    GPTOss120B,
		Name:                  "Together – GPT-OSS 120B",
		Provider:              "together",
		APIModel:              "openai/gpt-oss-120b",
		Currency:              "USD",
		CostPer1MIn:           0.15,
		CostPer1MOut:          0.6,
		ContextWindow:         128000,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GPTOss20B: {
		ID:                    GPTOss20B,
		Name:                  "Together – GPT-OSS 20B",
		Provider:              "together",
		APIModel:              "openai/gpt-oss-20b",
		Currency:              "USD",
		CostPer1MIn:           0.05,
		CostPer1MOut:          0.2,
		ContextWindow:         128000,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GLM51: {
		ID:                    GLM51,
		Name:                  "Together – GLM 5.1",
		Provider:              "together",
		APIModel:              "zai-org/GLM-5.1",
		Currency:              "USD",
		CostPer1MIn:           1.4,
		CostPer1MOut:          4.4,
		ContextWindow:         202752,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GLM52: {
		ID:                    GLM52,
		Name:                  "Together – GLM 5.2",
		Provider:              "together",
		APIModel:              "zai-org/GLM-5.2",
		Currency:              "USD",
		CostPer1MIn:           1.4,
		CostPer1MOut:          4.4,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
}
