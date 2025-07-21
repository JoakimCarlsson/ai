package model

type (
	ModelID       string
	ModelProvider string
)

type Model struct {
	ID                    ModelID       `json:"id"`
	Name                  string        `json:"name"`
	Provider              ModelProvider `json:"provider"`
	APIModel              string        `json:"api_model"`
	CostPer1MIn           float64       `json:"cost_per_1m_in"`
	CostPer1MOut          float64       `json:"cost_per_1m_out"`
	CostPer1MInCached     float64       `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached    float64       `json:"cost_per_1m_out_cached"`
	ContextWindow         int64         `json:"context_window"`
	DefaultMaxTokens      int64         `json:"default_max_tokens"`
	CanReason             bool          `json:"can_reason"`
	SupportsAttachments   bool          `json:"supports_attachments"`
	SupportsStructuredOut bool          `json:"supports_structured_output"`
}
