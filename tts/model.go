package tts

// AudioModel represents an audio generation (TTS) model with its
// configuration and capabilities.
//
// Each TTS provider package publishes its own catalog of these under the name
// Models, so a configuration is selected as, for example,
// elevenlabs.Models[elevenlabs.MultilingualV2].
type AudioModel struct {
	// ID is the unique identifier for this audio model.
	ID string `json:"id"`
	// Name is the human-readable name of the audio model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider string `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
	// Currency is the ISO 4217 code the cost fields are denominated in, for
	// example "USD" or "EUR". An empty value means "USD".
	Currency string `json:"currency"`
	// CostPer1MChars is the cost per 1 million characters, in Currency.
	CostPer1MChars float64 `json:"cost_per_1m_chars"`
	// MaxCharacters is the maximum number of characters per request.
	MaxCharacters int64 `json:"max_characters"`
	// SupportedFormats lists the audio formats this model can generate.
	SupportedFormats []string `json:"supported_formats,omitempty"`
	// DefaultFormat is the default audio format if not specified.
	DefaultFormat string `json:"default_format,omitempty"`
	// SupportsStreaming indicates if the model supports streaming audio
	// generation.
	SupportsStreaming bool `json:"supports_streaming"`
	// LatencyMs is the typical latency in milliseconds for audio generation.
	LatencyMs int64 `json:"latency_ms,omitempty"`
}
