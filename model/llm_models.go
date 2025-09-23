// Package model defines AI model configurations, providers, and capabilities.
//
// This package contains model definitions for all supported AI providers including
// OpenAI, Anthropic, Google, AWS Bedrock, Azure, and others. It provides detailed
// information about each model's capabilities, costs, context windows, and features.
//
// The package organizes models by provider and type (LLM, embedding, reranker),
// making it easy to select appropriate models for specific use cases while
// maintaining consistent configuration across the library.
//
// Key types include:
//   - Model for LLM configurations with pricing and capability information
//   - EmbeddingModel for text and multimodal embedding models
//   - RerankerModel for document reranking models
//   - Provider constants for identifying AI service providers
//
// Example usage:
//
//	// Get a specific model configuration
//	gpt4 := model.OpenAIModels[model.GPT4o]
//	fmt.Printf("Model: %s, Cost per 1M input tokens: $%.2f\n", gpt4.Name, gpt4.CostPer1MIn)
//
//	// Check model capabilities
//	if gpt4.SupportsStructuredOut {
//		fmt.Println("This model supports structured output")
//	}
package model

type (
	// ModelID is a unique identifier for a specific AI model.
	ModelID string
	// ModelProvider identifies the AI service provider (OpenAI, Anthropic, etc.).
	ModelProvider string
)

// Model represents a Large Language Model with its configuration and capabilities.
type Model struct {
	// ID is the unique identifier for this model within the library.
	ID ModelID `json:"id"`
	// Name is the human-readable name of the model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider ModelProvider `json:"provider"`
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
}
