package model

// ImageGenerationModel represents an image generation model with its configuration and capabilities.
type ImageGenerationModel struct {
	// ID is the unique identifier for this image generation model.
	ID ModelID `json:"id"`
	// Name is the human-readable name of the image generation model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider ModelProvider `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
	// CostPerImage is the cost per generated image in USD.
	CostPerImage float64 `json:"cost_per_image"`
	// MaxPromptTokens is the maximum number of tokens allowed in the prompt.
	MaxPromptTokens int64 `json:"max_prompt_tokens"`
	// SupportedSizes lists the image dimensions this model can generate.
	SupportedSizes []string `json:"supported_sizes,omitempty"`
	// DefaultSize is the default image size if not specified.
	DefaultSize string `json:"default_size,omitempty"`
}
