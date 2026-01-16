package model

const (
	ProviderElevenLabs ModelProvider = "elevenlabs"

	ElevenMultilingualV2 ModelID = "eleven_multilingual_v2"
	ElevenTurboV2_5      ModelID = "eleven_turbo_v2_5"
	ElevenFlashV2_5      ModelID = "eleven_flash_v2_5"
	ElevenTurboV3        ModelID = "eleven_turbo_v3"
)

// AudioModel represents an audio generation model with its configuration and capabilities.
type AudioModel struct {
	// ID is the unique identifier for this audio model.
	ID ModelID `json:"id"`
	// Name is the human-readable name of the audio model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider ModelProvider `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
	// CostPer1MChars is the cost per 1 million characters in USD.
	CostPer1MChars float64 `json:"cost_per_1m_chars"`
	// MaxCharacters is the maximum number of characters per request.
	MaxCharacters int64 `json:"max_characters"`
	// SupportedFormats lists the audio formats this model can generate.
	SupportedFormats []string `json:"supported_formats,omitempty"`
	// DefaultFormat is the default audio format if not specified.
	DefaultFormat string `json:"default_format,omitempty"`
	// SupportsStreaming indicates if the model supports streaming audio generation.
	SupportsStreaming bool `json:"supports_streaming"`
	// LatencyMs is the typical latency in milliseconds for audio generation.
	LatencyMs int64 `json:"latency_ms,omitempty"`
}

var ElevenLabsAudioModels = map[ModelID]AudioModel{
	ElevenMultilingualV2: {
		ID:                ElevenMultilingualV2,
		Name:              "Eleven Multilingual v2",
		Provider:          ProviderElevenLabs,
		APIModel:          "eleven_multilingual_v2",
		SupportedFormats:  []string{"mp3_44100_128", "mp3_44100_192", "pcm_16000", "pcm_22050", "pcm_24000", "pcm_44100"},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenTurboV2_5: {
		ID:                ElevenTurboV2_5,
		Name:              "Eleven Turbo v2.5",
		Provider:          ProviderElevenLabs,
		APIModel:          "eleven_turbo_v2_5",
		SupportedFormats:  []string{"mp3_44100_128", "mp3_44100_192", "pcm_16000", "pcm_22050", "pcm_24000", "pcm_44100"},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenFlashV2_5: {
		ID:                ElevenFlashV2_5,
		Name:              "Eleven Flash v2.5",
		Provider:          ProviderElevenLabs,
		APIModel:          "eleven_flash_v2_5",
		SupportedFormats:  []string{"mp3_44100_128", "mp3_44100_192", "pcm_16000", "pcm_22050", "pcm_24000", "pcm_44100"},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenTurboV3: {
		ID:                ElevenTurboV3,
		Name:              "Eleven Turbo v3",
		Provider:          ProviderElevenLabs,
		APIModel:          "eleven_turbo_v3",
		SupportedFormats:  []string{"mp3_44100_128", "mp3_44100_192", "pcm_16000", "pcm_22050", "pcm_24000", "pcm_44100"},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
}
