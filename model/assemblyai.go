package model

// ProviderAssemblyAI identifies the AssemblyAI speech-to-text provider.
const ProviderAssemblyAI Provider = "assemblyai"

// AssemblyAI transcription model IDs.
const (
	AssemblyAIBest                      ID = "best"
	AssemblyAINano                      ID = "nano"
	AssemblyAIUniversalStreamingEnglish ID = "universal-streaming-english"
	AssemblyAIUniversalStreamingMulti   ID = "universal-streaming-multilingual"
	AssemblyAIWhisperRT                 ID = "whisper-rt"
	AssemblyAIAlphaEnglish              ID = "alpha-english"
	AssemblyAIU3RTPro                   ID = "u3-rt-pro"
	AssemblyAIU3RTAgent                 ID = "u3-rt-agent"
)

// AssemblyAITranscriptionModels maps AssemblyAI model IDs to their
// configurations.
//
// Streaming model entries cover the v3 Universal Streaming endpoint
// (wss://streaming.assemblyai.com/v3/ws). Streaming pricing per AssemblyAI's
// public pricing page: Universal-Streaming $0.15/hr, Whisper-RT $0.30/hr,
// Universal-3 Pro Streaming $0.45/hr (CostPer1MIn here is the per-minute
// equivalent). Streaming endpoints accept pcm_s16le and pcm_mulaw at
// configurable sample rates with audio chunks of 50–1000 ms.
var AssemblyAITranscriptionModels = map[ID]TranscriptionModel{
	AssemblyAIBest: {
		ID:            AssemblyAIBest,
		Name:          "AssemblyAI Best",
		Provider:      ProviderAssemblyAI,
		APIModel:      "best",
		CostPer1MIn:   0.0035,
		MaxFileSizeMB: 5000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      false,
		SupportedResponseFormats: []string{
			"json", "text", "srt", "vtt",
		},
	},
	AssemblyAINano: {
		ID:            AssemblyAINano,
		Name:          "AssemblyAI Nano",
		Provider:      ProviderAssemblyAI,
		APIModel:      "nano",
		CostPer1MIn:   0.0025,
		MaxFileSizeMB: 5000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      false,
		SupportedResponseFormats: []string{
			"json", "text", "srt", "vtt",
		},
	},
	AssemblyAIUniversalStreamingEnglish: {
		ID:          AssemblyAIUniversalStreamingEnglish,
		Name:        "AssemblyAI Universal-Streaming English",
		Provider:    ProviderAssemblyAI,
		APIModel:    "universal-streaming-english",
		CostPer1MIn: 0.0025,
		SupportedFormats: []string{
			"pcm_s16le", "pcm_mulaw",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	AssemblyAIUniversalStreamingMulti: {
		ID:          AssemblyAIUniversalStreamingMulti,
		Name:        "AssemblyAI Universal-Streaming Multilingual",
		Provider:    ProviderAssemblyAI,
		APIModel:    "universal-streaming-multilingual",
		CostPer1MIn: 0.0025,
		SupportedFormats: []string{
			"pcm_s16le", "pcm_mulaw",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	AssemblyAIWhisperRT: {
		ID:          AssemblyAIWhisperRT,
		Name:        "AssemblyAI Whisper Realtime",
		Provider:    ProviderAssemblyAI,
		APIModel:    "whisper-rt",
		CostPer1MIn: 0.005,
		SupportedFormats: []string{
			"pcm_s16le", "pcm_mulaw",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	AssemblyAIAlphaEnglish: {
		ID:          AssemblyAIAlphaEnglish,
		Name:        "AssemblyAI Alpha English",
		Provider:    ProviderAssemblyAI,
		APIModel:    "alpha-english",
		CostPer1MIn: 0.0025,
		SupportedFormats: []string{
			"pcm_s16le", "pcm_mulaw",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	AssemblyAIU3RTPro: {
		ID:          AssemblyAIU3RTPro,
		Name:        "AssemblyAI Universal-3 Realtime Pro",
		Provider:    ProviderAssemblyAI,
		APIModel:    "u3-rt-pro",
		CostPer1MIn: 0.0075,
		SupportedFormats: []string{
			"pcm_s16le", "pcm_mulaw",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	AssemblyAIU3RTAgent: {
		ID:          AssemblyAIU3RTAgent,
		Name:        "AssemblyAI Universal-3 Realtime Agent",
		Provider:    ProviderAssemblyAI,
		APIModel:    "u3-rt-agent",
		CostPer1MIn: 0.0075,
		SupportedFormats: []string{
			"pcm_s16le", "pcm_mulaw",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
}
