package fireworks

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Fireworks provider identifier and a curated set of hosted chat model IDs
// for this registry.
//
// Fireworks hosts far more models than are catalogued here; callers can pass
// any "accounts/fireworks/models/..." path via [llmopenai.WithModel] even
// without a registered entry. Reasoning-class entries (DeepSeek R1, Qwen3
// thinking variants, GPT-OSS) carry Fireworks' "Logic / Long Context"
// surcharge with split input/output rates.
const (
	Llama31_70B          string = "fireworks.llama-v3p1-70b-instruct"
	Llama33_70B          string = "fireworks.llama-v3p3-70b-instruct"
	DeepSeekV3           string = "fireworks.deepseek-v3"
	DeepSeekV3p1Terminus string = "fireworks.deepseek-v3p1-terminus"
	DeepSeekR1           string = "fireworks.deepseek-r1"
	DeepSeekV4Pro        string = "fireworks.deepseek-v4-pro"
	DeepSeekV4Flash      string = "fireworks.deepseek-v4-flash"
	Qwen25_72B           string = "fireworks.qwen2p5-72b-instruct"
	Qwen3_235BInstruct   string = "fireworks.qwen3-235b-a22b-instruct-2507"
	Qwen3_30BThinking    string = "fireworks.qwen3-30b-a3b-thinking-2507"
	Qwen37Plus           string = "fireworks.qwen3p7-plus"
	Mixtral8x22B         string = "fireworks.mixtral-8x22b-instruct"
	KimiK2               string = "fireworks.kimi-k2-instruct"
	KimiK2_6             string = "fireworks.kimi-k2p6"
	KimiK2_7Code         string = "fireworks.kimi-k2p7-code"
	GPTOss120B           string = "fireworks.gpt-oss-120b"
	GPTOss20B            string = "fireworks.gpt-oss-20b"
	GLM52                string = "fireworks.glm-5p2"
	GLM51                string = "fireworks.glm-5p1"
	MiniMaxM3            string = "fireworks.minimax-m3"
	MiniMaxM2_7          string = "fireworks.minimax-m2p7"
	Nemotron3Ultra       string = "fireworks.nemotron-3-ultra"
)

// Models maps Fireworks model IDs to their configurations.
//
// Pricing source: https://docs.fireworks.ai/serverless/pricing.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Llama31_70B: {
		ID:                    Llama31_70B,
		Name:                  "Fireworks – Llama 3.1 70B Instruct",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/llama-v3p1-70b-instruct",
		Currency:              "USD",
		CostPer1MIn:           0.9,
		CostPer1MOut:          0.9,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	Llama33_70B: {
		ID:                    Llama33_70B,
		Name:                  "Fireworks – Llama 3.3 70B Instruct",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/llama-v3p3-70b-instruct",
		Currency:              "USD",
		CostPer1MIn:           0.9,
		CostPer1MOut:          0.9,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	DeepSeekV3: {
		ID:                    DeepSeekV3,
		Name:                  "Fireworks – DeepSeek V3",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/deepseek-v3",
		Currency:              "USD",
		CostPer1MIn:           0.56,
		CostPer1MOut:          1.68,
		ContextWindow:         163840,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	DeepSeekV3p1Terminus: {
		ID:                    DeepSeekV3p1Terminus,
		Name:                  "Fireworks – DeepSeek V3.1 Terminus",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/deepseek-v3p1-terminus",
		Currency:              "USD",
		CostPer1MIn:           0.56,
		CostPer1MOut:          1.68,
		ContextWindow:         163840,
		DefaultMaxTokens:      8192,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	DeepSeekR1: {
		ID:                    DeepSeekR1,
		Name:                  "Fireworks – DeepSeek R1",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/deepseek-r1",
		Currency:              "USD",
		CostPer1MIn:           3,
		CostPer1MOut:          8,
		ContextWindow:         163840,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	DeepSeekV4Pro: {
		ID:                    DeepSeekV4Pro,
		Name:                  "Fireworks – DeepSeek V4 Pro",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/deepseek-v4-pro",
		Currency:              "USD",
		CostPer1MIn:           1.74,
		CostPer1MOut:          3.48,
		ContextWindow:         1048576,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	DeepSeekV4Flash: {
		ID:                    DeepSeekV4Flash,
		Name:                  "Fireworks – DeepSeek V4 Flash",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/deepseek-v4-flash",
		Currency:              "USD",
		CostPer1MIn:           0.14,
		CostPer1MOut:          0.28,
		ContextWindow:         1048576,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen25_72B: {
		ID:                    Qwen25_72B,
		Name:                  "Fireworks – Qwen 2.5 72B Instruct",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/qwen2p5-72b-instruct",
		Currency:              "USD",
		CostPer1MIn:           0.9,
		CostPer1MOut:          0.9,
		ContextWindow:         32768,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	Qwen3_235BInstruct: {
		ID:                    Qwen3_235BInstruct,
		Name:                  "Fireworks – Qwen 3 235B Instruct (2507)",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/qwen3-235b-a22b-instruct-2507",
		Currency:              "USD",
		CostPer1MIn:           0.22,
		CostPer1MOut:          0.88,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		SupportsStructuredOut: true,
	},
	Qwen3_30BThinking: {
		ID:                    Qwen3_30BThinking,
		Name:                  "Fireworks – Qwen 3 30B A3B Thinking (2507)",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/qwen3-30b-a3b-thinking-2507",
		Currency:              "USD",
		CostPer1MIn:           0.15,
		CostPer1MOut:          0.6,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen37Plus: {
		ID:                    Qwen37Plus,
		Name:                  "Fireworks – Qwen 3.7 Plus",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/qwen3p7-plus",
		Currency:              "USD",
		CostPer1MIn:           0.4,
		CostPer1MOut:          1.6,
		ContextWindow:         1048576,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Mixtral8x22B: {
		ID:                    Mixtral8x22B,
		Name:                  "Fireworks – Mixtral 8x22B Instruct",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/mixtral-8x22b-instruct",
		Currency:              "USD",
		CostPer1MIn:           1.2,
		CostPer1MOut:          1.2,
		ContextWindow:         65536,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	KimiK2: {
		ID:                    KimiK2,
		Name:                  "Fireworks – Kimi K2 Instruct",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/kimi-k2-instruct",
		Currency:              "USD",
		CostPer1MIn:           0.6,
		CostPer1MOut:          3,
		ContextWindow:         131072,
		DefaultMaxTokens:      16384,
		SupportsStructuredOut: true,
	},
	KimiK2_6: {
		ID:                    KimiK2_6,
		Name:                  "Fireworks – Kimi K2.6",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/kimi-k2p6",
		Currency:              "USD",
		CostPer1MIn:           0.95,
		CostPer1MOut:          4,
		ContextWindow:         262144,
		DefaultMaxTokens:      16384,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	KimiK2_7Code: {
		ID:                    KimiK2_7Code,
		Name:                  "Fireworks – Kimi K2.7 Code",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/kimi-k2p7-code",
		Currency:              "USD",
		CostPer1MIn:           0.95,
		CostPer1MOut:          4,
		ContextWindow:         262144,
		DefaultMaxTokens:      16384,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GPTOss120B: {
		ID:                    GPTOss120B,
		Name:                  "Fireworks – GPT-OSS 120B",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/gpt-oss-120b",
		Currency:              "USD",
		CostPer1MIn:           0.15,
		CostPer1MOut:          0.6,
		ContextWindow:         131072,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GPTOss20B: {
		ID:                    GPTOss20B,
		Name:                  "Fireworks – GPT-OSS 20B",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/gpt-oss-20b",
		Currency:              "USD",
		CostPer1MIn:           0.07,
		CostPer1MOut:          0.3,
		ContextWindow:         131072,
		DefaultMaxTokens:      65536,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GLM52: {
		ID:                    GLM52,
		Name:                  "Fireworks – GLM 5.2",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/glm-5p2",
		Currency:              "USD",
		CostPer1MIn:           1.4,
		CostPer1MOut:          4.4,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GLM51: {
		ID:                    GLM51,
		Name:                  "Fireworks – GLM 5.1",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/glm-5p1",
		Currency:              "USD",
		CostPer1MIn:           1.4,
		CostPer1MOut:          4.4,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	MiniMaxM3: {
		ID:                    MiniMaxM3,
		Name:                  "Fireworks – MiniMax M3",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/minimax-m3",
		Currency:              "USD",
		CostPer1MIn:           0.3,
		CostPer1MOut:          1.2,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	MiniMaxM2_7: {
		ID:                    MiniMaxM2_7,
		Name:                  "Fireworks – MiniMax M2.7",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/minimax-m2p7",
		Currency:              "USD",
		CostPer1MIn:           0.3,
		CostPer1MOut:          1.2,
		ContextWindow:         262144,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Nemotron3Ultra: {
		ID:                    Nemotron3Ultra,
		Name:                  "Fireworks – NVIDIA Nemotron 3 Ultra",
		Provider:              "fireworks",
		APIModel:              "accounts/fireworks/models/nemotron-3-ultra",
		Currency:              "USD",
		CostPer1MIn:           0.6,
		CostPer1MOut:          2.4,
		ContextWindow:         131072,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
}
