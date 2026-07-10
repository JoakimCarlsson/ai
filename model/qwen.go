package model

// Qwen provider identifier and model IDs for this registry.
const (
	ProviderQwen Provider = "qwen"

	Qwen37Max      ID = "qwen-3.7-max"
	Qwen37Plus     ID = "qwen-3.7-plus"
	Qwen36Flash    ID = "qwen-3.6-flash"
	Qwen3Max       ID = "qwen-3-max"
	Qwen3Coder480B ID = "qwen-3-coder-480b"
	Qwen3CoderPlus ID = "qwen-3-coder-plus"
)

// QwenModels maps Qwen model IDs to their configurations.
var QwenModels = map[ID]Model{
	Qwen37Max: {
		ID:                    Qwen37Max,
		Name:                  "Qwen 3.7 Max",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3.7-max",
		CostPer1MIn:           2.50,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          7.50,
		ContextWindow:         1_000_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Qwen37Plus: {
		ID:                    Qwen37Plus,
		Name:                  "Qwen 3.7 Plus",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3.7-plus",
		CostPer1MIn:           0.40,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          1.60,
		ContextWindow:         1_000_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   true,
		SupportsStructuredOut: false,
	},
	Qwen36Flash: {
		ID:                    Qwen36Flash,
		Name:                  "Qwen 3.6 Flash",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3.6-flash",
		CostPer1MIn:           0.25,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          1.50,
		ContextWindow:         1_000_000,
		DefaultMaxTokens:      50000,
		CanReason:             true,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Qwen3Max: {
		ID:                    Qwen3Max,
		Name:                  "Qwen 3 Max",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3-max",
		CostPer1MIn:           1.20,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          6.00,
		ContextWindow:         256_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Qwen3Coder480B: {
		ID:                    Qwen3Coder480B,
		Name:                  "Qwen 3 Coder 480B",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3-coder-480b",
		CostPer1MIn:           1.50,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          7.50,
		ContextWindow:         256_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
	Qwen3CoderPlus: {
		ID:                    Qwen3CoderPlus,
		Name:                  "Qwen 3 Coder Plus",
		Provider:              ProviderQwen,
		APIModel:              "qwen-3-coder-plus",
		CostPer1MIn:           1.00,
		CostPer1MInCached:     0,
		CostPer1MOutCached:    0,
		CostPer1MOut:          5.00,
		ContextWindow:         1_000_000,
		DefaultMaxTokens:      50000,
		SupportsAttachments:   false,
		SupportsStructuredOut: false,
	},
}
