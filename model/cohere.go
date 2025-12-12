package model

const (
	ProviderCohere ModelProvider = "cohere"

	CommandRPlus ModelID = "command-r-plus"
	CommandR     ModelID = "command-r"
)

var CohereModels = map[ModelID]Model{
	CommandRPlus: {
		ID:                    CommandRPlus,
		Name:                  "Command R+",
		Provider:              ProviderCohere,
		APIModel:              "command-r-plus",
		CostPer1MIn:           2.50,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          10.00,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	CommandR: {
		ID:                    CommandR,
		Name:                  "Command R",
		Provider:              ProviderCohere,
		APIModel:              "command-r",
		CostPer1MIn:           0.15,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          0.60,
		ContextWindow:         128_000,
		DefaultMaxTokens:      4096,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
