package model

// ProviderGoogleCloud is the Google Cloud provider identifier for non-Gemini services.
const (
	ProviderGoogleCloud Provider = "google-cloud"

	GoogleCloudSTTDefault ID = "google-cloud-stt-default"
	GoogleCloudSTTLong    ID = "google-cloud-stt-long"

	GoogleCloudTTSStandard ID = "google-cloud-tts-standard"
	GoogleCloudTTSWavenet  ID = "google-cloud-tts-wavenet"
	GoogleCloudTTSNeural2  ID = "google-cloud-tts-neural2"
)

// GoogleCloudTranscriptionModels maps Google Cloud STT model IDs to their configurations.
var GoogleCloudTranscriptionModels = map[ID]TranscriptionModel{
	GoogleCloudSTTDefault: {
		ID:            GoogleCloudSTTDefault,
		Name:          "Google Cloud STT Default",
		Provider:      ProviderGoogleCloud,
		APIModel:      "default",
		CostPer1MIn:   0.016,
		MaxFileSizeMB: 480,
		SupportedFormats: []string{
			"flac", "linear16", "mulaw",
			"amr", "amr-wb", "ogg-opus",
			"speex", "webm-opus", "mp3",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    false,
		SupportsTranslation:    false,
		SupportsStreaming:      false,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	GoogleCloudSTTLong: {
		ID:            GoogleCloudSTTLong,
		Name:          "Google Cloud STT Long",
		Provider:      ProviderGoogleCloud,
		APIModel:      "long",
		CostPer1MIn:   0.016,
		MaxFileSizeMB: 480,
		SupportedFormats: []string{
			"flac", "linear16", "mulaw",
			"amr", "amr-wb", "ogg-opus",
			"speex", "webm-opus", "mp3",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    false,
		SupportsTranslation:    false,
		SupportsStreaming:      false,
		SupportedResponseFormats: []string{
			"json",
		},
	},
}

// GoogleCloudAudioModels maps Google Cloud TTS model IDs to their configurations.
var GoogleCloudAudioModels = map[ID]AudioModel{
	GoogleCloudTTSStandard: {
		ID:             GoogleCloudTTSStandard,
		Name:           "Google Cloud TTS Standard",
		Provider:       ProviderGoogleCloud,
		APIModel:       "standard",
		CostPer1MChars: 4.00,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16", "MP3", "OGG_OPUS",
			"MULAW", "ALAW",
		},
		DefaultFormat:     "MP3",
		SupportsStreaming: false,
	},
	GoogleCloudTTSWavenet: {
		ID:             GoogleCloudTTSWavenet,
		Name:           "Google Cloud TTS WaveNet",
		Provider:       ProviderGoogleCloud,
		APIModel:       "wavenet",
		CostPer1MChars: 16.00,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16", "MP3", "OGG_OPUS",
			"MULAW", "ALAW",
		},
		DefaultFormat:     "MP3",
		SupportsStreaming: false,
	},
	GoogleCloudTTSNeural2: {
		ID:             GoogleCloudTTSNeural2,
		Name:           "Google Cloud TTS Neural2",
		Provider:       ProviderGoogleCloud,
		APIModel:       "neural2",
		CostPer1MChars: 16.00,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16", "MP3", "OGG_OPUS",
			"MULAW", "ALAW",
		},
		DefaultFormat:     "MP3",
		SupportsStreaming: false,
	},
}
