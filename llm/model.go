package llm

// Model represents a Large Language Model with its configuration and
// capabilities.
//
// Each LLM provider package publishes its own catalog of these under the name
// Models, so a configuration is selected as, for example,
// openai.Models[openai.GPT4o]. A model that is not in a catalog can still be
// used by constructing one directly or via [NewCustomModel].
type Model struct {
	// ID is the unique identifier for this model within the library.
	ID string `json:"id"`
	// Name is the human-readable name of the model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider string `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
	// CostPer1MIn is the cost per 1 million input tokens in USD.
	CostPer1MIn float64 `json:"cost_per_1m_in"`
	// CostPer1MOut is the cost per 1 million output tokens in USD.
	CostPer1MOut float64 `json:"cost_per_1m_out"`
	// CostPer1MInCached is the cost per 1 million cached input tokens in USD.
	CostPer1MInCached float64 `json:"cost_per_1m_in_cached"`
	// CostPer1MOutCached is the cost per 1 million cached output tokens in USD.
	CostPer1MOutCached float64 `json:"cost_per_1m_out_cached"`
	// ContextWindow is the maximum number of tokens the model can process.
	ContextWindow int64 `json:"context_window"`
	// DefaultMaxTokens is the recommended maximum tokens for responses.
	DefaultMaxTokens int64 `json:"default_max_tokens"`
	// CanReason indicates if the model supports chain-of-thought reasoning.
	CanReason bool `json:"can_reason"`
	// SupportsAttachments indicates if the model can process images and files.
	SupportsAttachments bool `json:"supports_attachments"`
	// SupportsStructuredOut indicates if the model supports structured JSON output.
	SupportsStructuredOut bool `json:"supports_structured_output"`
	// SupportsImageGeneration indicates if the model can generate images.
	SupportsImageGeneration bool `json:"supports_image_generation"`
}
