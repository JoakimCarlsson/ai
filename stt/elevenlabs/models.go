package elevenlabs

import (
	"github.com/joakimcarlsson/ai/stt"
)

// ElevenLabs speech-to-text (Scribe) model IDs.
const (
	ScribeV1         string = "scribe_v1"
	ScribeV2         string = "scribe_v2"
	ScribeV2Realtime string = "scribe_v2_realtime"
)

// Models maps ElevenLabs Scribe model IDs to their
// configurations. Scribe v2 also exposes a Realtime WebSocket endpoint
// (wss://api.elevenlabs.io/v1/speech-to-text/realtime) accepting PCM at
// 8/16/22.05/24/44.1/48 kHz or μ-law 8 kHz, base64-encoded inside JSON
// input_audio_chunk events.
//
// Pricing source: https://elevenlabs.io/pricing/api.
// Fetched: 2026-07-26.
var Models = map[string]stt.TranscriptionModel{
	ScribeV1: {
		ID:            ScribeV1,
		Name:          "ElevenLabs Scribe v1",
		Provider:      "elevenlabs",
		APIModel:      "scribe_v1",
		CostPer1MIn:   0.0067,
		MaxFileSizeMB: 3000,
		SupportedFormats: []string{
			"mp3",
			"mp4",
			"wav",
			"flac",
			"ogg",
			"webm",
			"m4a",
			"aac",
			"aiff",
			"opus",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsDiarization:      true,
		SupportedResponseFormats: []string{"json", "text", "srt"},
	},
	ScribeV2: {
		ID:            ScribeV2,
		Name:          "ElevenLabs Scribe v2",
		Provider:      "elevenlabs",
		APIModel:      "scribe_v2",
		CostPer1MIn:   0.00367,
		MaxFileSizeMB: 3000,
		SupportedFormats: []string{
			"mp3",
			"mp4",
			"wav",
			"flac",
			"ogg",
			"webm",
			"m4a",
			"aac",
			"aiff",
			"opus",
			"pcm_8000",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
			"pcm_48000",
			"ulaw_8000",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsDiarization:      true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json", "text", "srt"},
	},
	ScribeV2Realtime: {
		ID:          ScribeV2Realtime,
		Name:        "ElevenLabs Scribe v2 Realtime",
		Provider:    "elevenlabs",
		APIModel:    "scribe_v2_realtime",
		CostPer1MIn: 0.0065,
		SupportedFormats: []string{
			"pcm_8000",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
			"pcm_48000",
			"ulaw_8000",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsDiarization:      true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json", "text", "srt"},
	},
}
