package azure

import (
	"github.com/joakimcarlsson/ai/stt"
)

// ProviderAzureSpeech is the Azure Speech Services provider identifier.
const (
	FastTranscription string = "azure-speech-fast-transcription"
	LLM               string = "azure-speech-llm"
)

// Models maps Azure Speech transcription model IDs to
// their configurations. Pricing source:
// https://azure.microsoft.com/pricing/details/cognitive-services/speech-services/
// Fetched: 2026-05-05.
var Models = map[string]stt.TranscriptionModel{
	FastTranscription: {
		ID:            FastTranscription,
		Name:          "Azure Speech Fast Transcription",
		Provider:      "azure-speech",
		APIModel:      "fast-transcription",
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
	},
	LLM: {
		ID:            LLM,
		Name:          "Azure Speech LLM",
		Provider:      "azure-speech",
		APIModel:      "llm-speech",
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
	},
}
