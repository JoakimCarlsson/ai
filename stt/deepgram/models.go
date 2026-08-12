package deepgram

import (
	"github.com/joakimcarlsson/ai/stt"
)

// Deepgram transcription model IDs.
const (
	Nova3       string = "nova-3"
	Nova2       string = "nova-2"
	FluxEnglish string = "flux-general-en"
	FluxMulti   string = "flux-general-multi"
)

// Models maps Deepgram model IDs to their
// configurations. Both Nova-3 and Nova-2 support batch (HTTP POST) and
// streaming (WebSocket wss://api.deepgram.com/v1/listen). Streaming accepts
// linear16 PCM among other encodings; CostPer1MIn is the per-minute price.
//
// Pricing source: https://deepgram.com/pricing.
// Fetched: 2026-07-26.
var Models = map[string]stt.TranscriptionModel{
	FluxEnglish: {
		ID:            FluxEnglish,
		Name:          "Deepgram Flux English",
		Provider:      "deepgram",
		APIModel:      "flux-general-en",
		CostPer1MIn:   0.0065,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3",
			"mp4",
			"wav",
			"flac",
			"ogg",
			"webm",
			"m4a",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	FluxMulti: {
		ID:            FluxMulti,
		Name:          "Deepgram Flux Multilingual",
		Provider:      "deepgram",
		APIModel:      "flux-general-multi",
		CostPer1MIn:   0.0078,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3",
			"mp4",
			"wav",
			"flac",
			"ogg",
			"webm",
			"m4a",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	Nova3: {
		ID:            Nova3,
		Name:          "Deepgram Nova 3",
		Provider:      "deepgram",
		APIModel:      "nova-3",
		CostPer1MIn:   0.0077,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3",
			"mp4",
			"wav",
			"flac",
			"ogg",
			"webm",
			"m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"vtt",
		},
	},
	Nova2: {
		ID:            Nova2,
		Name:          "Deepgram Nova 2",
		Provider:      "deepgram",
		APIModel:      "nova-2",
		CostPer1MIn:   0.0058,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3",
			"mp4",
			"wav",
			"flac",
			"ogg",
			"webm",
			"m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"vtt",
		},
	},
}
