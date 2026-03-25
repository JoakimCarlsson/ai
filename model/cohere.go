package model

// Cohere provider identifier and Command model IDs for this registry.
const (
	ProviderCohere Provider = "cohere"

	CommandRPlus ID = "command-r-plus"
	CommandR     ID = "command-r"
)

// CohereModels maps Cohere model IDs to their configurations.
var CohereModels = map[ID]Model{
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
