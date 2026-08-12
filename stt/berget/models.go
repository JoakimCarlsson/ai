package berget

import (
	"github.com/joakimcarlsson/ai/stt"
)

// Berget AI (https://berget.ai) is a Swedish, EU-hosted inference provider with
// an OpenAI-compatible API at https://api.berget.ai/v1.
//
// All prices below are in EUR, not USD: Berget bills in EUR and the Cost*
// fields hold the raw EUR figures from the /v1/models API (fetched 2026-06-30).
// The API does not return context windows, so ContextWindow values come from
// the upstream model cards (131_072 where a model's window is unpublished).
const (
	KBWhisperLarge       string = "KBLab/kb-whisper-large"
	NBWhisperLarge       string = "NbAiLab/nb-whisper-large"
	FasterWhisperLargeV3 string = "Systran/faster-whisper-large-v3"
)

// Models maps Berget speech-to-text model IDs to their
// configurations.
//
// Pricing source: https://api.berget.ai/v1/models.
// Fetched: 2026-07-26.
//
// Berget bills transcription at EUR 0.000033 / audio second; the
// TranscriptionModel struct has no per-second field, so CostPer1MIn holds the
// per-minute equivalent (0.000033 * 60), matching the AssemblyAI convention in
// this package.
var Models = map[string]stt.TranscriptionModel{
	KBWhisperLarge: {
		ID:          KBWhisperLarge,
		Name:        "KB Whisper Large (Swedish)",
		Provider:    "berget",
		APIModel:    "KBLab/kb-whisper-large",
		CostPer1MIn: 0.00198,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:  true,
		SupportsTranslation: true,
	},
	NBWhisperLarge: {
		ID:          NBWhisperLarge,
		Name:        "NB Whisper Large (Norwegian)",
		Provider:    "berget",
		APIModel:    "NbAiLab/nb-whisper-large",
		CostPer1MIn: 0.00198,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:  true,
		SupportsTranslation: true,
	},
	FasterWhisperLargeV3: {
		ID:          FasterWhisperLargeV3,
		Name:        "Faster Whisper Large v3",
		Provider:    "berget",
		APIModel:    "Systran/faster-whisper-large-v3",
		CostPer1MIn: 0.00198,
		SupportedFormats: []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		},
		SupportsTimestamps:  true,
		SupportsTranslation: true,
	},
}
