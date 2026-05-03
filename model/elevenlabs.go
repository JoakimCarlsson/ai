package model

// ProviderElevenLabs identifies the ElevenLabs speech and transcription
// provider.
const ProviderElevenLabs Provider = "elevenlabs"

// ElevenLabs text-to-speech model IDs.
const (
	ElevenV3             ID = "eleven_v3"
	ElevenMultilingualV2 ID = "eleven_multilingual_v2"
	ElevenFlashV2_5      ID = "eleven_flash_v2_5"
	ElevenFlashV2        ID = "eleven_flash_v2"
	ElevenTurboV2_5      ID = "eleven_turbo_v2_5"
	ElevenTurboV2        ID = "eleven_turbo_v2"
)

// ElevenLabs speech-to-text (Scribe) model IDs.
const (
	ElevenLabsScribeV1 ID = "scribe_v1"
	ElevenLabsScribeV2 ID = "scribe_v2"
)

// ElevenLabsAudioModels maps ElevenLabs speech model IDs to audio
// configurations.
var ElevenLabsAudioModels = map[ID]AudioModel{
	ElevenV3: {
		ID:            ElevenV3,
		Name:          "Eleven v3",
		Provider:      ProviderElevenLabs,
		APIModel:      "eleven_v3",
		MaxCharacters: 5000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenMultilingualV2: {
		ID:            ElevenMultilingualV2,
		Name:          "Eleven Multilingual v2",
		Provider:      ProviderElevenLabs,
		APIModel:      "eleven_multilingual_v2",
		MaxCharacters: 10000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenFlashV2_5: {
		ID:            ElevenFlashV2_5,
		Name:          "Eleven Flash v2.5",
		Provider:      ProviderElevenLabs,
		APIModel:      "eleven_flash_v2_5",
		MaxCharacters: 40000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenFlashV2: {
		ID:            ElevenFlashV2,
		Name:          "Eleven Flash v2",
		Provider:      ProviderElevenLabs,
		APIModel:      "eleven_flash_v2",
		MaxCharacters: 30000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenTurboV2_5: {
		ID:            ElevenTurboV2_5,
		Name:          "Eleven Turbo v2.5",
		Provider:      ProviderElevenLabs,
		APIModel:      "eleven_turbo_v2_5",
		MaxCharacters: 40000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	ElevenTurboV2: {
		ID:            ElevenTurboV2,
		Name:          "Eleven Turbo v2",
		Provider:      ProviderElevenLabs,
		APIModel:      "eleven_turbo_v2",
		MaxCharacters: 30000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
}

// ElevenLabsTranscriptionModels maps ElevenLabs Scribe model IDs to their
// configurations. Scribe v2 also exposes a Realtime WebSocket endpoint
// (wss://api.elevenlabs.io/v1/speech-to-text/realtime) accepting PCM at
// 8/16/22.05/24/44.1/48 kHz or μ-law 8 kHz, base64-encoded inside JSON
// input_audio_chunk events.
var ElevenLabsTranscriptionModels = map[ID]TranscriptionModel{
	ElevenLabsScribeV1: {
		ID:            ElevenLabsScribeV1,
		Name:          "ElevenLabs Scribe v1",
		Provider:      ProviderElevenLabs,
		APIModel:      "scribe_v1",
		CostPer1MIn:   0.0067,
		MaxFileSizeMB: 3000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a", "aac",
			"aiff", "opus",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      false,
		SupportedResponseFormats: []string{
			"json", "text", "srt",
		},
	},
	ElevenLabsScribeV2: {
		ID:            ElevenLabsScribeV2,
		Name:          "ElevenLabs Scribe v2",
		Provider:      ProviderElevenLabs,
		APIModel:      "scribe_v2",
		CostPer1MIn:   0.0067,
		MaxFileSizeMB: 3000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a", "aac",
			"aiff", "opus",
			"pcm_8000", "pcm_16000", "pcm_22050",
			"pcm_24000", "pcm_44100", "pcm_48000",
			"ulaw_8000",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json", "text", "srt",
		},
	},
}
