package google

import (
	"github.com/joakimcarlsson/ai/tts"
)

// ProviderGoogleCloud is the Google Cloud provider identifier for non-Gemini services.
const (
	TTSStandard string = "google-cloud-tts-standard"
	TTSWavenet  string = "google-cloud-tts-wavenet"
	TTSNeural2  string = "google-cloud-tts-neural2"
	TTSStudio   string = "google-cloud-tts-studio"
	TTSChirp3HD string = "google-cloud-tts-chirp3-hd"
)

// Models maps Google Cloud TTS model IDs to their configurations.
//
// Pricing source: https://cloud.google.com/text-to-speech/pricing.
// Fetched: not re-verified in the 2026-07-26 sweep.
var Models = map[string]tts.AudioModel{
	TTSStandard: {
		ID:             TTSStandard,
		Name:           "Google Cloud TTS Standard",
		Provider:       "google-cloud",
		APIModel:       "standard",
		Currency:       "USD",
		CostPer1MChars: 4,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16",
			"MP3",
			"OGG_OPUS",
			"MULAW",
			"ALAW",
		},
		DefaultFormat: "MP3",
	},
	TTSWavenet: {
		ID:             TTSWavenet,
		Name:           "Google Cloud TTS WaveNet",
		Provider:       "google-cloud",
		APIModel:       "wavenet",
		Currency:       "USD",
		CostPer1MChars: 16,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16",
			"MP3",
			"OGG_OPUS",
			"MULAW",
			"ALAW",
		},
		DefaultFormat: "MP3",
	},
	TTSNeural2: {
		ID:             TTSNeural2,
		Name:           "Google Cloud TTS Neural2",
		Provider:       "google-cloud",
		APIModel:       "neural2",
		Currency:       "USD",
		CostPer1MChars: 16,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16",
			"MP3",
			"OGG_OPUS",
			"MULAW",
			"ALAW",
		},
		DefaultFormat: "MP3",
	},
	TTSStudio: {
		ID:             TTSStudio,
		Name:           "Google Cloud TTS Studio",
		Provider:       "google-cloud",
		APIModel:       "studio",
		Currency:       "USD",
		CostPer1MChars: 160,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16",
			"MP3",
			"OGG_OPUS",
			"MULAW",
			"ALAW",
		},
		DefaultFormat: "MP3",
	},
	TTSChirp3HD: {
		ID:             TTSChirp3HD,
		Name:           "Google Cloud TTS Chirp 3: HD",
		Provider:       "google-cloud",
		APIModel:       "chirp3-hd",
		Currency:       "USD",
		CostPer1MChars: 30,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"LINEAR16",
			"MP3",
			"OGG_OPUS",
			"MULAW",
			"ALAW",
		},
		DefaultFormat:     "MP3",
		SupportsStreaming: true,
	},
}
