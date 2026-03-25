package model

// Perplexity provider identifier and Sonar model IDs for this registry.
const (
	ProviderPerplexity Provider = "perplexity"

	Sonar             ID = "sonar"
	SonarPro          ID = "sonar-pro"
	SonarReasoning    ID = "sonar-reasoning"
	SonarReasoningPro ID = "sonar-reasoning-pro"
	SonarDeepResearch ID = "sonar-deep-research"
)

// PerplexityModels maps Perplexity model IDs to their configurations.
var PerplexityModels = map[ID]Model{
	Sonar: {
		ID:                    Sonar,
		Name:                  "Sonar",
		Provider:              ProviderPerplexity,
		APIModel:              "sonar",
		CostPer1MIn:           1.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          1.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		CanReason:             false,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	SonarPro: {
		ID:                    SonarPro,
		Name:                  "Sonar Pro",
		Provider:              ProviderPerplexity,
		APIModel:              "sonar-pro",
		CostPer1MIn:           3.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          15.00,
		ContextWindow:         200_000,
		DefaultMaxTokens:      50000,
		CanReason:             false,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	SonarReasoning: {
		ID:                    SonarReasoning,
		Name:                  "Sonar Reasoning",
		Provider:              ProviderPerplexity,
		APIModel:              "sonar-reasoning",
		CostPer1MIn:           1.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          5.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	SonarReasoningPro: {
		ID:                    SonarReasoningPro,
		Name:                  "Sonar Reasoning Pro",
		Provider:              ProviderPerplexity,
		APIModel:              "sonar-reasoning-pro",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          8.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	SonarDeepResearch: {
		ID:                    SonarDeepResearch,
		Name:                  "Sonar Deep Research",
		Provider:              ProviderPerplexity,
		APIModel:              "sonar-deep-research",
		CostPer1MIn:           2.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          8.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
