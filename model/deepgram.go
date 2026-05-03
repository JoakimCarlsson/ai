package model

// ProviderDeepgram identifies the Deepgram speech-to-text and text-to-speech
// provider.
const ProviderDeepgram Provider = "deepgram"

// Deepgram transcription model IDs.
const (
	DeepgramNova3 ID = "nova-3"
	DeepgramNova2 ID = "nova-2"
)

// Deepgram Aura text-to-speech model/voice IDs. Deepgram's TTS model
// identifier encodes voice and language together (format:
// aura[-2]-<voice>-<lang>).
const (
	DeepgramAura2Thalia    ID = "aura-2-thalia-en"
	DeepgramAura2Andromeda ID = "aura-2-andromeda-en"
	DeepgramAura2Helena    ID = "aura-2-helena-en"
	DeepgramAuraAsteria    ID = "aura-asteria-en"
	DeepgramAuraLuna       ID = "aura-luna-en"
	DeepgramAuraStella     ID = "aura-stella-en"
	DeepgramAuraZeus       ID = "aura-zeus-en"
)

// DeepgramAudioModels maps Deepgram Aura TTS model IDs to their
// configurations. Each model identifies a specific voice + language; pick
// the variant that matches the desired voice. Aura-2 models offer lower
// latency (~90ms TTFB) and higher quality at a higher per-character cost.
var DeepgramAudioModels = map[ID]AudioModel{
	DeepgramAura2Thalia: {
		ID:             DeepgramAura2Thalia,
		Name:           "Deepgram Aura-2 Thalia",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-thalia-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Andromeda: {
		ID:             DeepgramAura2Andromeda,
		Name:           "Deepgram Aura-2 Andromeda",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-andromeda-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Helena: {
		ID:             DeepgramAura2Helena,
		Name:           "Deepgram Aura-2 Helena",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-helena-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraAsteria: {
		ID:             DeepgramAuraAsteria,
		Name:           "Deepgram Aura Asteria",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-asteria-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraLuna: {
		ID:             DeepgramAuraLuna,
		Name:           "Deepgram Aura Luna",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-luna-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraStella: {
		ID:             DeepgramAuraStella,
		Name:           "Deepgram Aura Stella",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-stella-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraZeus: {
		ID:             DeepgramAuraZeus,
		Name:           "Deepgram Aura Zeus",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-zeus-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
}

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
