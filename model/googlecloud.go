package model

// ProviderGoogleCloud is the Google Cloud provider identifier for non-Gemini services.
const (
	ProviderGoogleCloud Provider = "google-cloud"

	GoogleCloudSTTDefault ID = "google-cloud-stt-default"
	GoogleCloudSTTLong    ID = "google-cloud-stt-long"
	GoogleCloudSTTChirp2  ID = "google-cloud-stt-chirp-2"
	GoogleCloudSTTChirp3  ID = "google-cloud-stt-chirp-3"

	GoogleCloudTTSStandard ID = "google-cloud-tts-standard"
	GoogleCloudTTSWavenet  ID = "google-cloud-tts-wavenet"
	GoogleCloudTTSNeural2  ID = "google-cloud-tts-neural2"
	GoogleCloudTTSStudio   ID = "google-cloud-tts-studio"
	GoogleCloudTTSChirp3HD ID = "google-cloud-tts-chirp3-hd"
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
	GoogleCloudSTTChirp2: {
		ID:            GoogleCloudSTTChirp2,
		Name:          "Google Cloud STT Chirp 2",
		Provider:      ProviderGoogleCloud,
		APIModel:      "chirp_2",
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
		SupportsTranslation:    true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	GoogleCloudSTTChirp3: {
		ID:            GoogleCloudSTTChirp3,
		Name:          "Google Cloud STT Chirp 3",
		Provider:      ProviderGoogleCloud,
		APIModel:      "chirp_3",
		CostPer1MIn:   0.016,
		MaxFileSizeMB: 480,
		SupportedFormats: []string{
			"flac", "linear16", "mulaw",
			"amr", "amr-wb", "ogg-opus",
			"speex", "webm-opus", "mp3",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: false,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
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
	GoogleCloudTTSStudio: {
		ID:             GoogleCloudTTSStudio,
		Name:           "Google Cloud TTS Studio",
		Provider:       ProviderGoogleCloud,
		APIModel:       "studio",
		CostPer1MChars: 160.00,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16", "MP3", "OGG_OPUS",
			"MULAW", "ALAW",
		},
		DefaultFormat:     "MP3",
		SupportsStreaming: false,
	},
	GoogleCloudTTSChirp3HD: {
		ID:             GoogleCloudTTSChirp3HD,
		Name:           "Google Cloud TTS Chirp 3: HD",
		Provider:       ProviderGoogleCloud,
		APIModel:       "chirp3-hd",
		CostPer1MChars: 30.00,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16", "MP3", "OGG_OPUS",
			"MULAW", "ALAW",
		},
		DefaultFormat:     "MP3",
		SupportsStreaming: true,
	},
}
