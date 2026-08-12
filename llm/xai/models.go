package xai

import (
	"github.com/joakimcarlsson/ai/llm"
)

// xAI provider identifier and Grok model IDs for this registry.
const (
	Grok4                  string = "grok-4-0709"
	Grok4FastReasoning     string = "grok-4-fast-reasoning"
	Grok4FastNonReasoning  string = "grok-4-fast-non-reasoning"
	Grok41FastReasoning    string = "grok-4-1-fast-reasoning"
	Grok41FastNonReasoning string = "grok-4-1-fast-non-reasoning"
	Grok3                  string = "grok-3"
	Grok3Mini              string = "grok-3-mini"
	Grok3Fast              string = "grok-3-fast"
	Grok3MiniFast          string = "grok-3-mini-fast"
	Grok2Vision            string = "grok-2-vision-1212"
	GrokCodeFast1          string = "grok-code-fast-1"
	Grok420Reasoning       string = "grok-4.20-0309-reasoning"
	Grok420NonReasoning    string = "grok-4.20-0309-non-reasoning"
	Grok420MultiAgent      string = "grok-4.20-multi-agent-0309"
	Grok43                 string = "grok-4.3"
	Grok45                 string = "grok-4.5"
	GrokBuild01            string = "grok-build-0.1"
)

// Models maps xAI chat model IDs to their configurations.
//
// Pricing source: https://docs.x.ai/docs/models.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Grok4: {
		ID:                    Grok4,
		Name:                  "Grok4",
		Provider:              "xai",
		APIModel:              "grok-4-0709",
		CostPer1MIn:           3,
		CostPer1MOut:          15,
		CostPer1MInCached:     0.75,
		ContextWindow:         256000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok3: {
		ID:                    Grok3,
		Name:                  "Grok3",
		Provider:              "xai",
		APIModel:              "grok-3",
		CostPer1MIn:           3,
		CostPer1MOut:          15,
		ContextWindow:         131072,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok3Mini: {
		ID:                    Grok3Mini,
		Name:                  "Grok3 Mini",
		Provider:              "xai",
		APIModel:              "grok-3-mini",
		CostPer1MIn:           0.3,
		CostPer1MOut:          0.5,
		ContextWindow:         131072,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok3Fast: {
		ID:                    Grok3Fast,
		Name:                  "Grok3 Fast",
		Provider:              "xai",
		APIModel:              "grok-3-fast",
		CostPer1MIn:           5,
		CostPer1MOut:          25,
		ContextWindow:         131072,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok3MiniFast: {
		ID:                    Grok3MiniFast,
		Name:                  "Grok3 Mini Fast",
		Provider:              "xai",
		APIModel:              "grok-3-mini-fast",
		CostPer1MIn:           0.6,
		CostPer1MOut:          4,
		ContextWindow:         131072,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok2Vision: {
		ID:                    Grok2Vision,
		Name:                  "Grok2 Vision",
		Provider:              "xai",
		APIModel:              "grok-2-vision-1212",
		CostPer1MIn:           2,
		CostPer1MOut:          10,
		ContextWindow:         32768,
		DefaultMaxTokens:      4000,
		SupportsStructuredOut: true,
	},
	Grok4FastReasoning: {
		ID:                    Grok4FastReasoning,
		Name:                  "Grok4 Fast Reasoning",
		Provider:              "xai",
		APIModel:              "grok-4-fast-reasoning",
		CostPer1MIn:           0.2,
		CostPer1MOut:          0.5,
		CostPer1MInCached:     0.05,
		ContextWindow:         2000000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok4FastNonReasoning: {
		ID:                    Grok4FastNonReasoning,
		Name:                  "Grok4 Fast Non-Reasoning",
		Provider:              "xai",
		APIModel:              "grok-4-fast-non-reasoning",
		CostPer1MIn:           0.2,
		CostPer1MOut:          0.5,
		CostPer1MInCached:     0.05,
		ContextWindow:         2000000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok41FastReasoning: {
		ID:                    Grok41FastReasoning,
		Name:                  "Grok4.1 Fast Reasoning",
		Provider:              "xai",
		APIModel:              "grok-4-1-fast-reasoning",
		CostPer1MIn:           0.2,
		CostPer1MOut:          0.5,
		CostPer1MInCached:     0.05,
		ContextWindow:         2000000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok41FastNonReasoning: {
		ID:                    Grok41FastNonReasoning,
		Name:                  "Grok4.1 Fast Non-Reasoning",
		Provider:              "xai",
		APIModel:              "grok-4-1-fast-non-reasoning",
		CostPer1MIn:           0.2,
		CostPer1MOut:          0.5,
		CostPer1MInCached:     0.05,
		ContextWindow:         2000000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	GrokCodeFast1: {
		ID:                    GrokCodeFast1,
		Name:                  "Grok Code Fast 1",
		Provider:              "xai",
		APIModel:              "grok-code-fast-1",
		CostPer1MIn:           0.2,
		CostPer1MOut:          1.5,
		CostPer1MInCached:     0.02,
		ContextWindow:         256000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok420Reasoning: {
		ID:                    Grok420Reasoning,
		Name:                  "Grok 4.20 Reasoning",
		Provider:              "xai",
		APIModel:              "grok-4.20-0309-reasoning",
		CostPer1MIn:           2,
		CostPer1MOut:          6,
		ContextWindow:         2000000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok420NonReasoning: {
		ID:                    Grok420NonReasoning,
		Name:                  "Grok 4.20 Non-Reasoning",
		Provider:              "xai",
		APIModel:              "grok-4.20-0309-non-reasoning",
		CostPer1MIn:           2,
		CostPer1MOut:          6,
		ContextWindow:         2000000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok420MultiAgent: {
		ID:                    Grok420MultiAgent,
		Name:                  "Grok 4.20 Multi-Agent",
		Provider:              "xai",
		APIModel:              "grok-4.20-multi-agent-0309",
		CostPer1MIn:           2,
		CostPer1MOut:          6,
		ContextWindow:         2000000,
		DefaultMaxTokens:      20000,
		SupportsStructuredOut: true,
	},
	Grok43: {
		ID:                    Grok43,
		Name:                  "Grok 4.3",
		Provider:              "xai",
		APIModel:              "grok-4.3",
		CostPer1MIn:           1.25,
		CostPer1MOut:          2.5,
		ContextWindow:         1000000,
		DefaultMaxTokens:      32000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	Grok45: {
		ID:                    Grok45,
		Name:                  "Grok 4.5",
		Provider:              "xai",
		APIModel:              "grok-4.5",
		CostPer1MIn:           2,
		CostPer1MOut:          6,
		CostPer1MInCached:     0.3,
		ContextWindow:         500000,
		DefaultMaxTokens:      32000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	GrokBuild01: {
		ID:                    GrokBuild01,
		Name:                  "Grok Build 0.1",
		Provider:              "xai",
		APIModel:              "grok-build-0.1",
		CostPer1MIn:           1,
		CostPer1MOut:          2,
		CostPer1MInCached:     0.2,
		ContextWindow:         256000,
		DefaultMaxTokens:      20000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
}
