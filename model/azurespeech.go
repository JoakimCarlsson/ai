package model

// ProviderAzureSpeech is the Azure Speech Services provider identifier.
const (
	ProviderAzureSpeech Provider = "azure-speech"

	AzureSpeechNeural            ID = "azure-speech-neural"
	AzureSpeechNeuralHD          ID = "azure-speech-neural-hd"
	AzureSpeechFastTranscription ID = "azure-speech-fast-transcription"
	AzureSpeechLLM               ID = "azure-speech-llm"
)

// AzureSpeechAudioModels maps Azure Speech model IDs to their configurations.
var AzureSpeechAudioModels = map[ID]AudioModel{
	AzureSpeechNeural: {
		ID:             AzureSpeechNeural,
		Name:           "Azure Speech Neural",
		Provider:       ProviderAzureSpeech,
		APIModel:       "neural",
		CostPer1MChars: 16.00,
		MaxCharacters:  10000,
		SupportedFormats: []string{
			"audio-16khz-128kbitrate-mono-mp3",
			"audio-24khz-160kbitrate-mono-mp3",
			"riff-16khz-16bit-mono-pcm",
			"riff-24khz-16bit-mono-pcm",
			"ogg-16khz-16bit-mono-opus",
			"ogg-24khz-16bit-mono-opus",
		},
		DefaultFormat:     "audio-24khz-160kbitrate-mono-mp3",
		SupportsStreaming: false,
	},
	AzureSpeechNeuralHD: {
		ID:             AzureSpeechNeuralHD,
		Name:           "Azure Speech Neural HD",
		Provider:       ProviderAzureSpeech,
		APIModel:       "neural-hd",
		CostPer1MChars: 22.00,
		MaxCharacters:  10000,
		SupportedFormats: []string{
			"audio-16khz-128kbitrate-mono-mp3",
			"audio-24khz-160kbitrate-mono-mp3",
			"audio-48khz-192kbitrate-mono-mp3",
			"riff-16khz-16bit-mono-pcm",
			"riff-24khz-16bit-mono-pcm",
			"riff-48khz-16bit-mono-pcm",
			"ogg-16khz-16bit-mono-opus",
			"ogg-24khz-16bit-mono-opus",
		},
		DefaultFormat:     "audio-24khz-160kbitrate-mono-mp3",
		SupportsStreaming: true,
	},
}

// AzureSpeechTranscriptionModels maps Azure Speech transcription model IDs to
// their configurations. Pricing source:
// https://azure.microsoft.com/pricing/details/cognitive-services/speech-services/
// Fetched: 2026-05-05.
var AzureSpeechTranscriptionModels = map[ID]TranscriptionModel{
	AzureSpeechFastTranscription: {
		ID:            AzureSpeechFastTranscription,
		Name:          "Azure Speech Fast Transcription",
		Provider:      ProviderAzureSpeech,
		APIModel:      "fast-transcription",
		CostPer1MIn:   0,
		CostPer1MOut:  0,
		MaxFileSizeMB: 200,
		SupportedFormats: []string{
			"wav",
			"mp3",
			"ogg",
			"flac",
			"wma",
			"aac",
			"alaw",
			"mulaw",
			"amr",
			"webm",
			"speex",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      false,
	},
	AzureSpeechLLM: {
		ID:            AzureSpeechLLM,
		Name:          "Azure Speech LLM",
		Provider:      ProviderAzureSpeech,
		APIModel:      "llm-speech",
		CostPer1MIn:   0,
		CostPer1MOut:  0,
		MaxFileSizeMB: 500,
		SupportedFormats: []string{
			"wav",
			"mp3",
			"ogg",
			"flac",
			"wma",
			"aac",
			"alaw",
			"mulaw",
			"amr",
			"webm",
			"speex",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    true,
		SupportsStreaming:      false,
	},
}
