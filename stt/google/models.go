package google

import (
	"github.com/joakimcarlsson/ai/stt"
)

// ProviderGoogleCloud is the Google Cloud provider identifier for non-Gemini services.
const (
	STTDefault string = "google-cloud-stt-default"
	STTLong    string = "google-cloud-stt-long"
	STTChirp2  string = "google-cloud-stt-chirp-2"
	STTChirp3  string = "google-cloud-stt-chirp-3"
)

// Models maps Google Cloud STT model IDs to their configurations.
//
// Pricing source: https://cloud.google.com/speech-to-text/pricing.
// Fetched: not re-verified in the 2026-07-26 sweep.
var Models = map[string]stt.TranscriptionModel{
	STTDefault: {
		ID:            STTDefault,
		Name:          "Google Cloud STT Default",
		Provider:      "google-cloud",
		APIModel:      "default",
		Currency:      "USD",
		CostPer1MIn:   0.016,
		MaxFileSizeMB: 480,
		SupportedFormats: []string{
			"flac",
			"linear16",
			"mulaw",
			"amr",
			"amr-wb",
			"ogg-opus",
			"speex",
			"webm-opus",
			"mp3",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportedResponseFormats: []string{"json"},
	},
	STTLong: {
		ID:            STTLong,
		Name:          "Google Cloud STT Long",
		Provider:      "google-cloud",
		APIModel:      "long",
		Currency:      "USD",
		CostPer1MIn:   0.016,
		MaxFileSizeMB: 480,
		SupportedFormats: []string{
			"flac",
			"linear16",
			"mulaw",
			"amr",
			"amr-wb",
			"ogg-opus",
			"speex",
			"webm-opus",
			"mp3",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportedResponseFormats: []string{"json"},
	},
	STTChirp2: {
		ID:            STTChirp2,
		Name:          "Google Cloud STT Chirp 2",
		Provider:      "google-cloud",
		APIModel:      "chirp_2",
		Currency:      "USD",
		CostPer1MIn:   0.016,
		MaxFileSizeMB: 480,
		SupportedFormats: []string{
			"flac",
			"linear16",
			"mulaw",
			"amr",
			"amr-wb",
			"ogg-opus",
			"speex",
			"webm-opus",
			"mp3",
		},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsTranslation:      true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	STTChirp3: {
		ID:            STTChirp3,
		Name:          "Google Cloud STT Chirp 3",
		Provider:      "google-cloud",
		APIModel:      "chirp_3",
		Currency:      "USD",
		CostPer1MIn:   0.016,
		MaxFileSizeMB: 480,
		SupportedFormats: []string{
			"flac",
			"linear16",
			"mulaw",
			"amr",
			"amr-wb",
			"ogg-opus",
			"speex",
			"webm-opus",
			"mp3",
		},
		SupportsTimestamps:       true,
		SupportsDiarization:      true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
}
