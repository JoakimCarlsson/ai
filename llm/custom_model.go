package llm

// NewCustomModel creates a Model configuration for custom or local models.
// This function enables BYOM (Bring Your Own Model) support, allowing external
// users to define models for Ollama, local endpoints, or custom API providers.
//
// Example usage with Ollama:
//
//	llama := llm.NewCustomModel(
//	    llm.WithModelID("llama3.2"),
//	    llm.WithAPIModel("llama3.2:latest"),
//	    llm.WithName("Llama 3.2"),
//	    llm.WithContextWindow(128_000),
//	    llm.WithStructuredOutput(true),
//	)
//
// Example with custom pricing:
//
//	custom := llm.NewCustomModel(
//	    llm.WithModelID("my-model"),
//	    llm.WithAPIModel("custom-v1"),
//	    llm.WithCostPer1MIn(1.50),
//	    llm.WithCostPer1MOut(5.00),
//	)
func NewCustomModel(opts ...ModelOption) Model {
	m := Model{
		Provider:                "custom",
		SupportsStructuredOut:   false,
		SupportsAttachments:     false,
		CanReason:               false,
		SupportsImageGeneration: false,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// ModelOption configures a custom Model when passed to NewCustomModel. It is
// distinct from the per-provider client options such as openai.Option.
type ModelOption func(*Model)

// WithModelID sets the unique identifier for the model.
func WithModelID(id string) ModelOption {
	return func(m *Model) {
		m.ID = id
	}
}

// WithName sets the human-readable name of the model.
func WithName(name string) ModelOption {
	return func(m *Model) {
		m.Name = name
	}
}

// WithProvider sets the provider identifier for the model.
func WithProvider(provider string) ModelOption {
	return func(m *Model) {
		m.Provider = provider
	}
}

// WithAPIModel sets the model identifier used in API requests.
func WithAPIModel(apiModel string) ModelOption {
	return func(m *Model) {
		m.APIModel = apiModel
	}
}

// WithCurrency sets the ISO 4217 code the cost fields are denominated in,
// for example "USD" or "EUR". An empty value means "USD".
func WithCurrency(currency string) ModelOption {
	return func(m *Model) {
		m.Currency = currency
	}
}

// WithCostPer1MIn sets the cost per 1 million input tokens, in the model's
// currency.
func WithCostPer1MIn(cost float64) ModelOption {
	return func(m *Model) {
		m.CostPer1MIn = cost
	}
}

// WithCostPer1MOut sets the cost per 1 million output tokens, in the model's
// currency.
func WithCostPer1MOut(cost float64) ModelOption {
	return func(m *Model) {
		m.CostPer1MOut = cost
	}
}

// WithCostPer1MInCached sets the cost per 1 million cached input tokens, in the
// model's currency.
func WithCostPer1MInCached(cost float64) ModelOption {
	return func(m *Model) {
		m.CostPer1MInCached = cost
	}
}

// WithCostPer1MOutCached sets the cost per 1 million cached output tokens, in
// the model's currency.
func WithCostPer1MOutCached(cost float64) ModelOption {
	return func(m *Model) {
		m.CostPer1MOutCached = cost
	}
}

// WithContextWindow sets the maximum number of tokens the model can process.
func WithContextWindow(window int64) ModelOption {
	return func(m *Model) {
		m.ContextWindow = window
	}
}

// WithDefaultMaxTokens sets the recommended maximum tokens for responses.
func WithDefaultMaxTokens(maxTokens int64) ModelOption {
	return func(m *Model) {
		m.DefaultMaxTokens = maxTokens
	}
}

// WithReasoning sets whether the model supports chain-of-thought reasoning.
func WithReasoning(canReason bool) ModelOption {
	return func(m *Model) {
		m.CanReason = canReason
	}
}

// WithAttachments sets whether the model can process images and files.
func WithAttachments(supportsAttachments bool) ModelOption {
	return func(m *Model) {
		m.SupportsAttachments = supportsAttachments
	}
}

// WithStructuredOutput sets whether the model supports structured JSON output.
func WithStructuredOutput(supportsStructuredOutput bool) ModelOption {
	return func(m *Model) {
		m.SupportsStructuredOut = supportsStructuredOutput
	}
}

// WithImageGeneration sets whether the model can generate images.
func WithImageGeneration(supportsImageGeneration bool) ModelOption {
	return func(m *Model) {
		m.SupportsImageGeneration = supportsImageGeneration
	}
}
