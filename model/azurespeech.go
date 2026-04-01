package model

// ProviderAzureSpeech is the Azure Speech Services provider identifier.
const (
	ProviderAzureSpeech Provider = "azure-speech"

	AzureSpeechNeural ID = "azure-speech-neural"
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
}
