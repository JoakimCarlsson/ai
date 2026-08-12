package elevenlabs

import (
	"github.com/joakimcarlsson/ai/tts"
)

// ElevenLabs text-to-speech model IDs.
const (
	V3             string = "eleven_v3"
	MultilingualV2 string = "eleven_multilingual_v2"
	FlashV2_5      string = "eleven_flash_v2_5"
	FlashV2        string = "eleven_flash_v2"
	TurboV2_5      string = "eleven_turbo_v2_5"
	TurboV2        string = "eleven_turbo_v2"
)

// Models maps ElevenLabs speech model IDs to audio
// configurations.
//
// Pricing source: https://elevenlabs.io/pricing/api.
// Fetched: 2026-07-26.
var Models = map[string]tts.AudioModel{
	V3: {
		ID:             V3,
		Name:           "Eleven v3",
		Provider:       "elevenlabs",
		APIModel:       "eleven_v3",
		Currency:       "USD",
		CostPer1MChars: 100,
		MaxCharacters:  5000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	MultilingualV2: {
		ID:             MultilingualV2,
		Name:           "Eleven Multilingual v2",
		Provider:       "elevenlabs",
		APIModel:       "eleven_multilingual_v2",
		Currency:       "USD",
		CostPer1MChars: 100,
		MaxCharacters:  10000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	FlashV2_5: {
		ID:             FlashV2_5,
		Name:           "Eleven Flash v2.5",
		Provider:       "elevenlabs",
		APIModel:       "eleven_flash_v2_5",
		Currency:       "USD",
		CostPer1MChars: 50,
		MaxCharacters:  40000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	FlashV2: {
		ID:             FlashV2,
		Name:           "Eleven Flash v2",
		Provider:       "elevenlabs",
		APIModel:       "eleven_flash_v2",
		Currency:       "USD",
		CostPer1MChars: 50,
		MaxCharacters:  30000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	TurboV2_5: {
		ID:             TurboV2_5,
		Name:           "Eleven Turbo v2.5",
		Provider:       "elevenlabs",
		APIModel:       "eleven_turbo_v2_5",
		Currency:       "USD",
		CostPer1MChars: 50,
		MaxCharacters:  40000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
	TurboV2: {
		ID:             TurboV2,
		Name:           "Eleven Turbo v2",
		Provider:       "elevenlabs",
		APIModel:       "eleven_turbo_v2",
		Currency:       "USD",
		CostPer1MChars: 50,
		MaxCharacters:  30000,
		SupportedFormats: []string{
			"mp3_44100_128",
			"mp3_44100_192",
			"pcm_16000",
			"pcm_22050",
			"pcm_24000",
			"pcm_44100",
		},
		DefaultFormat:     "mp3_44100_128",
		SupportsStreaming: true,
	},
}
