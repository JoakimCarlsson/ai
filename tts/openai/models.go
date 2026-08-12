package openai

import (
	"github.com/joakimcarlsson/ai/tts"
)

// OpenAI text-to-speech model IDs.
const (
	TTS1    string = "tts-1"
	TTS1HD  string = "tts-1-hd"
	MiniTTS string = "gpt-4o-mini-tts"
)

// Models maps OpenAI TTS model IDs to their configurations.
//
// Pricing source: https://developers.openai.com/api/docs/pricing (TTS rates
// not published on the page; carried forward).
// Fetched: not re-verified in the 2026-07-26 sweep.
var Models = map[string]tts.AudioModel{
	TTS1: {
		ID:             TTS1,
		Name:           "OpenAI TTS-1",
		Provider:       "openai",
		APIModel:       "tts-1",
		Currency:       "USD",
		CostPer1MChars: 15,
		MaxCharacters:  4096,
		SupportedFormats: []string{
			"mp3",
			"opus",
			"aac",
			"flac",
			"wav",
			"pcm",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	TTS1HD: {
		ID:             TTS1HD,
		Name:           "OpenAI TTS-1 HD",
		Provider:       "openai",
		APIModel:       "tts-1-hd",
		Currency:       "USD",
		CostPer1MChars: 30,
		MaxCharacters:  4096,
		SupportedFormats: []string{
			"mp3",
			"opus",
			"aac",
			"flac",
			"wav",
			"pcm",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	MiniTTS: {
		ID:             MiniTTS,
		Name:           "GPT-4o Mini TTS",
		Provider:       "openai",
		APIModel:       "gpt-4o-mini-tts",
		Currency:       "USD",
		CostPer1MChars: 12,
		MaxCharacters:  4096,
		SupportedFormats: []string{
			"mp3",
			"opus",
			"aac",
			"flac",
			"wav",
			"pcm",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
}
