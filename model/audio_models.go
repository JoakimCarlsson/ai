package model

// OpenAI text-to-speech model IDs.
const (
	OpenAITTS1    ID = "tts-1"
	OpenAITTS1HD  ID = "tts-1-hd"
	OpenAIMiniTTS ID = "gpt-4o-mini-tts"
)

// OpenAI speech-to-text model IDs.
const (
	Whisper1                ID = "whisper-1"
	GPT4oTranscribe         ID = "gpt-4o-transcribe"
	GPT4oMiniTranscribe     ID = "gpt-4o-mini-transcribe"
	GPT4oMiniTranscribe2025 ID = "gpt-4o-mini-transcribe-2025-12-15"
	GPT4oTranscribeDiarize  ID = "gpt-4o-transcribe-diarize"
)

// AudioModel represents an audio generation (TTS) model with its
// configuration and capabilities.
type AudioModel struct {
	// ID is the unique identifier for this audio model.
	ID ID `json:"id"`
	// Name is the human-readable name of the audio model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider Provider `json:"provider"`
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
	// SupportsStreaming indicates if the model supports streaming audio
	// generation.
	SupportsStreaming bool `json:"supports_streaming"`
	// LatencyMs is the typical latency in milliseconds for audio generation.
	LatencyMs int64 `json:"latency_ms,omitempty"`
}

// TranscriptionModel represents a speech-to-text transcription model with
// its configuration and capabilities.
type TranscriptionModel struct {
	ID                       ID       `json:"id"`
	Name                     string   `json:"name"`
	Provider                 Provider `json:"provider"`
	APIModel                 string   `json:"api_model"`
	CostPer1MIn              float64  `json:"cost_per_1m_in"`
	CostPer1MOut             float64  `json:"cost_per_1m_out"`
	MaxFileSizeMB            int64    `json:"max_file_size_mb"`
	SupportedFormats         []string `json:"supported_formats,omitempty"`
	SupportsTimestamps       bool     `json:"supports_timestamps"`
	SupportsWordTimestamps   bool     `json:"supports_word_timestamps"`
	SupportsDiarization      bool     `json:"supports_diarization"`
	SupportsTranslation      bool     `json:"supports_translation"`
	SupportsStreaming        bool     `json:"supports_streaming"`
	SupportedResponseFormats []string `json:"supported_response_formats,omitempty"`
}

// OpenAITranscriptionModels contains configuration for OpenAI speech-to-text
// models.
var OpenAITranscriptionModels = map[ID]TranscriptionModel{
	Whisper1: {
		ID:            Whisper1,
		Name:          "Whisper v2",
		Provider:      ProviderOpenAI,
		APIModel:      "whisper-1",
		CostPer1MIn:   0.006,
		MaxFileSizeMB: 25,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    false,
		SupportsTranslation:    true,
		SupportsStreaming:      false,
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"verbose_json",
			"vtt",
		},
	},
	GPT4oTranscribe: {
		ID:            GPT4oTranscribe,
		Name:          "GPT-4o Transcribe",
		Provider:      ProviderOpenAI,
		APIModel:      "gpt-4o-transcribe",
		CostPer1MIn:   0.10,
		CostPer1MOut:  0.40,
		MaxFileSizeMB: 25,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:       false,
		SupportsWordTimestamps:   false,
		SupportsDiarization:      false,
		SupportsTranslation:      false,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	GPT4oMiniTranscribe: {
		ID:            GPT4oMiniTranscribe,
		Name:          "GPT-4o Mini Transcribe",
		Provider:      ProviderOpenAI,
		APIModel:      "gpt-4o-mini-transcribe",
		CostPer1MIn:   0.02,
		CostPer1MOut:  0.08,
		MaxFileSizeMB: 25,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:       false,
		SupportsWordTimestamps:   false,
		SupportsDiarization:      false,
		SupportsTranslation:      false,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	GPT4oMiniTranscribe2025: {
		ID:            GPT4oMiniTranscribe2025,
		Name:          "GPT-4o Mini Transcribe 2025-12-15",
		Provider:      ProviderOpenAI,
		APIModel:      "gpt-4o-mini-transcribe-2025-12-15",
		CostPer1MIn:   0.02,
		CostPer1MOut:  0.08,
		MaxFileSizeMB: 25,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:       false,
		SupportsWordTimestamps:   false,
		SupportsDiarization:      false,
		SupportsTranslation:      false,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	GPT4oTranscribeDiarize: {
		ID:            GPT4oTranscribeDiarize,
		Name:          "GPT-4o Transcribe Diarize",
		Provider:      ProviderOpenAI,
		APIModel:      "gpt-4o-transcribe-diarize",
		MaxFileSizeMB: 25,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   false,
		SupportsDiarization:      true,
		SupportsTranslation:      false,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json", "text", "diarized_json"},
	},
}

// OpenAIAudioModels maps OpenAI TTS model IDs to their configurations.
var OpenAIAudioModels = map[ID]AudioModel{
	OpenAITTS1: {
		ID:             OpenAITTS1,
		Name:           "OpenAI TTS-1",
		Provider:       ProviderOpenAI,
		APIModel:       "tts-1",
		CostPer1MChars: 15.00,
		MaxCharacters:  4096,
		SupportedFormats: []string{
			"mp3",
			"opus",
			"aac",
			"flac",
			"wav",
			"pcm",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	OpenAITTS1HD: {
		ID:             OpenAITTS1HD,
		Name:           "OpenAI TTS-1 HD",
		Provider:       ProviderOpenAI,
		APIModel:       "tts-1-hd",
		CostPer1MChars: 30.00,
		MaxCharacters:  4096,
		SupportedFormats: []string{
			"mp3",
			"opus",
			"aac",
			"flac",
			"wav",
			"pcm",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	OpenAIMiniTTS: {
		ID:             OpenAIMiniTTS,
		Name:           "GPT-4o Mini TTS",
		Provider:       ProviderOpenAI,
		APIModel:       "gpt-4o-mini-tts",
		CostPer1MChars: 12.00,
		MaxCharacters:  4096,
		SupportedFormats: []string{
			"mp3",
			"opus",
			"aac",
			"flac",
			"wav",
			"pcm",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
}
