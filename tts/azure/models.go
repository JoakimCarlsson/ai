package azure

import (
	"github.com/joakimcarlsson/ai/tts"
)

// ProviderAzureSpeech is the Azure Speech Services provider identifier.
const (
	Neural   string = "azure-speech-neural"
	NeuralHD string = "azure-speech-neural-hd"
)

// Models maps Azure Speech model IDs to their configurations.
//
// Pricing source:
// https://azure.microsoft.com/pricing/details/cognitive-services/speech-services/.
// Fetched: not re-verified in the 2026-07-26 sweep.
var Models = map[string]tts.AudioModel{
	Neural: {
		ID:             Neural,
		Name:           "Azure Speech Neural",
		Provider:       "azure-speech",
		APIModel:       "neural",
		CostPer1MChars: 16,
		MaxCharacters:  10000,
		SupportedFormats: []string{
			"audio-16khz-128kbitrate-mono-mp3",
			"audio-24khz-160kbitrate-mono-mp3",
			"riff-16khz-16bit-mono-pcm",
			"riff-24khz-16bit-mono-pcm",
			"ogg-16khz-16bit-mono-opus",
			"ogg-24khz-16bit-mono-opus",
		},
		DefaultFormat: "audio-24khz-160kbitrate-mono-mp3",
	},
	NeuralHD: {
		ID:             NeuralHD,
		Name:           "Azure Speech Neural HD",
		Provider:       "azure-speech",
		APIModel:       "neural-hd",
		CostPer1MChars: 22,
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
