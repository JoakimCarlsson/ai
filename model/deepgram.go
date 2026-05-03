package model

// ProviderDeepgram identifies the Deepgram speech-to-text provider.
const ProviderDeepgram Provider = "deepgram"

// Deepgram transcription model IDs.
const (
	DeepgramNova3 ID = "nova-3"
	DeepgramNova2 ID = "nova-2"
)

// DeepgramTranscriptionModels maps Deepgram model IDs to their
// configurations. Both Nova-3 and Nova-2 support batch (HTTP POST) and
// streaming (WebSocket wss://api.deepgram.com/v1/listen). Streaming accepts
// linear16 PCM among other encodings; CostPer1MIn is the per-minute price.
var DeepgramTranscriptionModels = map[ID]TranscriptionModel{
	DeepgramNova3: {
		ID:            DeepgramNova3,
		Name:          "Deepgram Nova 3",
		Provider:      ProviderDeepgram,
		APIModel:      "nova-3",
		CostPer1MIn:   0.0077,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json", "text", "srt", "vtt",
		},
	},
	DeepgramNova2: {
		ID:            DeepgramNova2,
		Name:          "Deepgram Nova 2",
		Provider:      ProviderDeepgram,
		APIModel:      "nova-2",
		CostPer1MIn:   0.0058,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json", "text", "srt", "vtt",
		},
	},
}
