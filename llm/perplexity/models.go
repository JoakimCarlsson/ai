package perplexity

import (
	"github.com/joakimcarlsson/ai/llm"
)

// Perplexity provider identifier and Sonar model IDs for this registry.
const (
	Sonar             string = "sonar"
	SonarPro          string = "sonar-pro"
	SonarReasoning    string = "sonar-reasoning"
	SonarReasoningPro string = "sonar-reasoning-pro"
	SonarDeepResearch string = "sonar-deep-research"
)

// Models maps Perplexity model IDs to their configurations.
//
// Pricing source: https://docs.perplexity.ai/getting-started/pricing.
// Fetched: 2026-07-26.
var Models = map[string]llm.Model{
	Sonar: {
		ID:               Sonar,
		Name:             "Sonar",
		Provider:         "perplexity",
		APIModel:         "sonar",
		Currency:         "USD",
		CostPer1MIn:      1,
		CostPer1MOut:     1,
		ContextWindow:    128000,
		DefaultMaxTokens: 50000,
	},
	SonarPro: {
		ID:               SonarPro,
		Name:             "Sonar Pro",
		Provider:         "perplexity",
		APIModel:         "sonar-pro",
		Currency:         "USD",
		CostPer1MIn:      3,
		CostPer1MOut:     15,
		ContextWindow:    200000,
		DefaultMaxTokens: 50000,
	},
	SonarReasoning: {
		ID:               SonarReasoning,
		Name:             "Sonar Reasoning",
		Provider:         "perplexity",
		APIModel:         "sonar-reasoning",
		Currency:         "USD",
		CostPer1MIn:      1,
		CostPer1MOut:     5,
		ContextWindow:    128000,
		DefaultMaxTokens: 50000,
		CanReason:        true,
	},
	SonarReasoningPro: {
		ID:               SonarReasoningPro,
		Name:             "Sonar Reasoning Pro",
		Provider:         "perplexity",
		APIModel:         "sonar-reasoning-pro",
		Currency:         "USD",
		CostPer1MIn:      2,
		CostPer1MOut:     8,
		ContextWindow:    128000,
		DefaultMaxTokens: 50000,
		CanReason:        true,
	},
	SonarDeepResearch: {
		ID:               SonarDeepResearch,
		Name:             "Sonar Deep Research",
		Provider:         "perplexity",
		APIModel:         "sonar-deep-research",
		Currency:         "USD",
		CostPer1MIn:      2,
		CostPer1MOut:     8,
		ContextWindow:    128000,
		DefaultMaxTokens: 50000,
		CanReason:        true,
	},
}
