package decision

// Model represents a decision model with its configuration and pricing.
//
// Each decision provider package publishes its own catalog of these under the
// name Models, so a configuration is selected as, for example,
// jev.Models[jev.JevLatest].
type Model struct {
	// ID is the unique identifier for this decision model.
	ID string `json:"id"`
	// Name is the human-readable name of the decision model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider string `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
	// Currency is the ISO 4217 code the cost fields are denominated in, for
	// example "USD" or "EUR". An empty value means "USD".
	Currency string `json:"currency"`
	// CostPer1MInputTokens is the cost per 1 million input tokens, in Currency.
	CostPer1MInputTokens float64 `json:"cost_per_1m_input_tokens"`
	// CostPer1MOutputTokens is the cost per 1 million output tokens, in Currency.
	CostPer1MOutputTokens float64 `json:"cost_per_1m_output_tokens"`
}
