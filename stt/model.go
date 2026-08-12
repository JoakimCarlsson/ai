package stt

// TranscriptionModel represents a speech-to-text transcription model with
// its configuration and capabilities.
//
// Each STT provider package publishes its own catalog of these under the name
// Models, so a configuration is selected as, for example,
// openai.Models[openai.Whisper1].
type TranscriptionModel struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	Provider                 string   `json:"provider"`
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
