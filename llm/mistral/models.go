package mistral

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Mistral provider identifier and model IDs for this registry.
const (
	Large3            string = "mistral-large-3"
	Medium35          string = "mistral-medium-3.5"
	Medium31          string = "mistral-medium-3.1"
	Small4            string = "mistral-small-4"
	Small32           string = "mistral-small-3.2"
	Ministral3_14B    string = "ministral-3-14b"
	Ministral3_8B     string = "ministral-3-8b"
	Ministral3_3B     string = "ministral-3-3b"
	Nemo              string = "mistral-nemo"
	Codestral         string = "codestral"
	Devstral2         string = "devstral-2"
	Medium3           string = "mistral-medium-3"
	Mixtral8x7B       string = "mixtral-8x7b"
	Mistral7B         string = "mistral-7b"
	MagistralMedium12 string = "magistral-medium-1.2"
	MagistralSmall12  string = "magistral-small-1.2"
)

// Models maps Mistral model IDs to their configurations.
//
// Pricing source: https://mistral.ai/pricing/api.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Large3: {
		ID:                    Large3,
		Name:                  "Mistral Large 3",
		Provider:              "mistral",
		APIModel:              "mistral-large-2512",
		Currency:              "USD",
		CostPer1MIn:           0.5,
		CostPer1MOut:          1.5,
		ContextWindow:         262144,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Medium35: {
		ID:                    Medium35,
		Name:                  "Mistral Medium 3.5",
		Provider:              "mistral",
		APIModel:              "mistral-medium-2604",
		Currency:              "USD",
		CostPer1MIn:           1.5,
		CostPer1MOut:          7.5,
		ContextWindow:         262144,
		DefaultMaxTokens:      8192,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Medium31: {
		ID:                    Medium31,
		Name:                  "Mistral Medium 3.1",
		Provider:              "mistral",
		APIModel:              "mistral-medium-2508",
		Currency:              "USD",
		CostPer1MIn:           0.4,
		CostPer1MOut:          2,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Small4: {
		ID:                    Small4,
		Name:                  "Mistral Small 4",
		Provider:              "mistral",
		APIModel:              "mistral-small-2603",
		Currency:              "USD",
		CostPer1MIn:           0.15,
		CostPer1MOut:          0.6,
		ContextWindow:         262144,
		DefaultMaxTokens:      8192,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Small32: {
		ID:                    Small32,
		Name:                  "Mistral Small 3.2",
		Provider:              "mistral",
		APIModel:              "mistral-small-2506",
		Currency:              "USD",
		CostPer1MIn:           0.06,
		CostPer1MOut:          0.18,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Ministral3_14B: {
		ID:                    Ministral3_14B,
		Name:                  "Ministral 3 14B",
		Provider:              "mistral",
		APIModel:              "ministral-14b-2512",
		Currency:              "USD",
		CostPer1MIn:           0.2,
		CostPer1MOut:          0.2,
		ContextWindow:         262144,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Ministral3_8B: {
		ID:                    Ministral3_8B,
		Name:                  "Ministral 3 8B",
		Provider:              "mistral",
		APIModel:              "ministral-8b-2512",
		Currency:              "USD",
		CostPer1MIn:           0.15,
		CostPer1MOut:          0.15,
		ContextWindow:         262144,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Ministral3_3B: {
		ID:                    Ministral3_3B,
		Name:                  "Ministral 3 3B",
		Provider:              "mistral",
		APIModel:              "ministral-3b-2512",
		Currency:              "USD",
		CostPer1MIn:           0.1,
		CostPer1MOut:          0.1,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Nemo: {
		ID:                    Nemo,
		Name:                  "Mistral Nemo",
		Provider:              "mistral",
		APIModel:              "open-mistral-nemo",
		Currency:              "USD",
		CostPer1MIn:           0.15,
		CostPer1MOut:          0.15,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	Codestral: {
		ID:                    Codestral,
		Name:                  "Codestral",
		Provider:              "mistral",
		APIModel:              "codestral-2508",
		Currency:              "USD",
		CostPer1MIn:           0.3,
		CostPer1MOut:          0.9,
		ContextWindow:         256000,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	Devstral2: {
		ID:                    Devstral2,
		Name:                  "Devstral 2",
		Provider:              "mistral",
		APIModel:              "devstral-2512",
		Currency:              "USD",
		CostPer1MIn:           0.4,
		CostPer1MOut:          2,
		ContextWindow:         262144,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	Medium3: {
		ID:                    Medium3,
		Name:                  "Mistral Medium 3",
		Provider:              "mistral",
		APIModel:              "mistral-medium-2505",
		Currency:              "USD",
		CostPer1MIn:           0.4,
		CostPer1MOut:          2,
		ContextWindow:         131072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Mixtral8x7B: {
		ID:               Mixtral8x7B,
		Name:             "Mixtral 8x7B",
		Provider:         "mistral",
		APIModel:         "mixtral-8x7b-instruct-v0.1",
		Currency:         "USD",
		CostPer1MIn:      0.7,
		CostPer1MOut:     0.7,
		ContextWindow:    32000,
		DefaultMaxTokens: 4096,
	},
	Mistral7B: {
		ID:               Mistral7B,
		Name:             "Mistral 7B Instruct",
		Provider:         "mistral",
		APIModel:         "mistral-7b-instruct-v0.3",
		Currency:         "USD",
		CostPer1MIn:      0.028,
		CostPer1MOut:     0.054,
		ContextWindow:    32768,
		DefaultMaxTokens: 4096,
	},
	MagistralMedium12: {
		ID:                    MagistralMedium12,
		Name:                  "Magistral Medium 1.2",
		Provider:              "mistral",
		APIModel:              "magistral-medium-2509",
		Currency:              "USD",
		CostPer1MIn:           2,
		CostPer1MOut:          5,
		ContextWindow:         128000,
		DefaultMaxTokens:      8192,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	MagistralSmall12: {
		ID:                    MagistralSmall12,
		Name:                  "Magistral Small 1.2",
		Provider:              "mistral",
		APIModel:              "magistral-small-2509",
		Currency:              "USD",
		CostPer1MIn:           0.5,
		CostPer1MOut:          1.5,
		ContextWindow:         128000,
		DefaultMaxTokens:      8192,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}
