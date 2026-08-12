package ollama

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Ollama provider identifier and a representative subset of locally-pulled
// open-weights model IDs for this registry.
//
// Ollama runs models locally; there is no API spend per request. The cost
// fields are zero across the board — real cost is electricity / GPU rental,
// not per-token API fees. Ollama supports any model the user has pulled
// locally; callers can pass any model id via [llmopenai.WithModel] even
// without a registered entry here.
const (
	Llama32_3B      string = "ollama.llama3.2:3b"
	Llama33_70B     string = "ollama.llama3.3:70b"
	Llama4Scout     string = "ollama.llama4:scout"
	Qwen25_7B       string = "ollama.qwen2.5:7b"
	Qwen25_72B      string = "ollama.qwen2.5:72b"
	Qwen25Coder7B   string = "ollama.qwen2.5-coder:7b"
	Qwen25Coder32B  string = "ollama.qwen2.5-coder:32b"
	Qwen3_8B        string = "ollama.qwen3:8b"
	Qwen3_32B       string = "ollama.qwen3:32b"
	DeepSeekR1_8B   string = "ollama.deepseek-r1:8b"
	DeepSeekR1_70B  string = "ollama.deepseek-r1:70b"
	Mistral7B       string = "ollama.mistral:7b"
	MistralSmall24B string = "ollama.mistral-small:24b"
	Gemma3_4B       string = "ollama.gemma3:4b"
	Gemma3_27B      string = "ollama.gemma3:27b"
	Phi4_14B        string = "ollama.phi4:14b"
	GptOss20B       string = "ollama.gpt-oss:20b"
	GptOss120B      string = "ollama.gpt-oss:120b"
)

// Models maps Ollama model IDs to their configurations.
//
// Local inference; all per-token costs are zero. See https://ollama.com/library
// for the full library. Fetched: 2026-05-04.
var Models = map[string]llm.Model{
	Llama32_3B: {
		ID:                    Llama32_3B,
		Name:                  "Ollama – Llama 3.2 3B",
		Provider:              "ollama",
		APIModel:              "llama3.2:3b",
		ContextWindow:         128000,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	Llama33_70B: {
		ID:                    Llama33_70B,
		Name:                  "Ollama – Llama 3.3 70B",
		Provider:              "ollama",
		APIModel:              "llama3.3:70b",
		ContextWindow:         128000,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	Qwen25_7B: {
		ID:                    Qwen25_7B,
		Name:                  "Ollama – Qwen 2.5 7B",
		Provider:              "ollama",
		APIModel:              "qwen2.5:7b",
		ContextWindow:         128000,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	Qwen25_72B: {
		ID:                    Qwen25_72B,
		Name:                  "Ollama – Qwen 2.5 72B",
		Provider:              "ollama",
		APIModel:              "qwen2.5:72b",
		ContextWindow:         128000,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	DeepSeekR1_8B: {
		ID:                    DeepSeekR1_8B,
		Name:                  "Ollama – DeepSeek R1 Distill 8B",
		Provider:              "ollama",
		APIModel:              "deepseek-r1:8b",
		ContextWindow:         128000,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	DeepSeekR1_70B: {
		ID:                    DeepSeekR1_70B,
		Name:                  "Ollama – DeepSeek R1 Distill 70B",
		Provider:              "ollama",
		APIModel:              "deepseek-r1:70b",
		ContextWindow:         128000,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Mistral7B: {
		ID:               Mistral7B,
		Name:             "Ollama – Mistral 7B",
		Provider:         "ollama",
		APIModel:         "mistral:7b",
		ContextWindow:    32768,
		DefaultMaxTokens: 4096,
	},
	Llama4Scout: {
		ID:                    Llama4Scout,
		Name:                  "Ollama – Llama 4 Scout",
		Provider:              "ollama",
		APIModel:              "llama4:scout",
		ContextWindow:         10000000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Qwen25Coder7B: {
		ID:                    Qwen25Coder7B,
		Name:                  "Ollama – Qwen 2.5 Coder 7B",
		Provider:              "ollama",
		APIModel:              "qwen2.5-coder:7b",
		ContextWindow:         32768,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	Qwen25Coder32B: {
		ID:                    Qwen25Coder32B,
		Name:                  "Ollama – Qwen 2.5 Coder 32B",
		Provider:              "ollama",
		APIModel:              "qwen2.5-coder:32b",
		ContextWindow:         32768,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	Qwen3_8B: {
		ID:                    Qwen3_8B,
		Name:                  "Ollama – Qwen 3 8B",
		Provider:              "ollama",
		APIModel:              "qwen3:8b",
		ContextWindow:         40000,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	Qwen3_32B: {
		ID:                    Qwen3_32B,
		Name:                  "Ollama – Qwen 3 32B",
		Provider:              "ollama",
		APIModel:              "qwen3:32b",
		ContextWindow:         40000,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	MistralSmall24B: {
		ID:                    MistralSmall24B,
		Name:                  "Ollama – Mistral Small 24B",
		Provider:              "ollama",
		APIModel:              "mistral-small:24b",
		ContextWindow:         32768,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	Gemma3_4B: {
		ID:                    Gemma3_4B,
		Name:                  "Ollama – Gemma 3 4B",
		Provider:              "ollama",
		APIModel:              "gemma3:4b",
		ContextWindow:         128000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Gemma3_27B: {
		ID:                    Gemma3_27B,
		Name:                  "Ollama – Gemma 3 27B",
		Provider:              "ollama",
		APIModel:              "gemma3:27b",
		ContextWindow:         128000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Phi4_14B: {
		ID:                    Phi4_14B,
		Name:                  "Ollama – Phi-4 14B",
		Provider:              "ollama",
		APIModel:              "phi4:14b",
		ContextWindow:         16000,
		DefaultMaxTokens:      4096,
		SupportsStructuredOut: true,
	},
	GptOss20B: {
		ID:                    GptOss20B,
		Name:                  "Ollama – GPT-OSS 20B",
		Provider:              "ollama",
		APIModel:              "gpt-oss:20b",
		ContextWindow:         128000,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
	GptOss120B: {
		ID:                    GptOss120B,
		Name:                  "Ollama – GPT-OSS 120B",
		Provider:              "ollama",
		APIModel:              "gpt-oss:120b",
		ContextWindow:         128000,
		DefaultMaxTokens:      32768,
		CanReason:             true,
		SupportsStructuredOut: true,
	},
}
