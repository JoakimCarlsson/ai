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
	// Pricing maps size to quality to cost per image in USD.
	// For models with fixed pricing (no quality levels), use a single quality key like "default".
	// Example: Pricing["1024x1024"]["standard"] = 0.04
	Pricing map[string]map[string]float64 `json:"pricing"`
	// MaxPromptTokens is the maximum number of tokens allowed in the prompt.
	MaxPromptTokens int64 `json:"max_prompt_tokens"`
	// SupportedSizes lists the image dimensions this model can generate.
	SupportedSizes []string `json:"supported_sizes,omitempty"`
	// DefaultSize is the default image size if not specified.
	DefaultSize string `json:"default_size,omitempty"`
	// SupportedQualities lists the quality levels this model supports (e.g., "standard", "hd", "low", "medium", "high").
	SupportedQualities []string `json:"supported_qualities,omitempty"`
	// DefaultQuality is the default quality level if not specified.
	DefaultQuality string `json:"default_quality,omitempty"`
}
