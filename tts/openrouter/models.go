package openrouter

import (
	"github.com/joakimcarlsson/ai/tts"
)

// OpenRouter text-to-speech model IDs.
const (
	MAIVoice2            string = "openrouter.mai-voice-2"
	MAIVoice2Flash       string = "openrouter.mai-voice-2-flash"
	VoxtralMiniTTS       string = "openrouter.voxtral-mini-tts-2603"
	GrokVoiceTTS1        string = "openrouter.grok-voice-tts-1.0"
	Aura2                string = "openrouter.aura-2"
	Gemini31FlashTTS     string = "openrouter.gemini-3.1-flash-tts-preview"
	QwenAudio3TTSFlash   string = "openrouter.qwen-audio-3.0-tts-flash"
	QwenAudio3TTSPlus    string = "openrouter.qwen-audio-3.0-tts-plus"
	FishAudioS1          string = "openrouter.s1"
	FishAudioS2Pro       string = "openrouter.s2-pro"
	FishAudioS21Pro      string = "openrouter.s2.1-pro"
	MiniMaxSpeech28HD    string = "openrouter.speech-2.8-hd"
	MiniMaxSpeech28Turbo string = "openrouter.speech-2.8-turbo"
	ZonosTransformer     string = "openrouter.zonos-v0.1-transformer"
	ZonosHybrid          string = "openrouter.zonos-v0.1-hybrid"
	Orpheus3B            string = "openrouter.orpheus-3b-0.1-ft"
	CSM1B                string = "openrouter.csm-1b"
	Kokoro82M            string = "openrouter.kokoro-82m"
)

// Models maps OpenRouter text-to-speech model IDs to their
// configurations.
//
// Known-good defaults; any OpenRouter speech model id works even without an
// entry here.
//
// Source: https://openrouter.ai/api/v1/models?output_modalities=speech, whose
// prompt rate for speech models is quoted per input character and is scaled to
// CostPer1MChars here. Fetched: 2026-07-31.
//
// OpenRouter's /audio/speech defaults to pcm where OpenAI defaults to mp3, so
// DefaultFormat records pcm. Voice ids are per-model and per-upstream; there is
// no list-voices route, so consult the model's page.
var Models = map[string]tts.AudioModel{
	MAIVoice2: {
		ID:               MAIVoice2,
		Name:             "OpenRouter – MAI-Voice-2",
		Provider:         "openrouter",
		APIModel:         "microsoft/mai-voice-2",
		CostPer1MChars:   22,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	MAIVoice2Flash: {
		ID:               MAIVoice2Flash,
		Name:             "OpenRouter – MAI-Voice-2 Flash",
		Provider:         "openrouter",
		APIModel:         "microsoft/mai-voice-2-flash",
		CostPer1MChars:   15,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	VoxtralMiniTTS: {
		ID:               VoxtralMiniTTS,
		Name:             "OpenRouter – Voxtral Mini TTS",
		Provider:         "openrouter",
		APIModel:         "mistralai/voxtral-mini-tts-2603",
		CostPer1MChars:   16,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	GrokVoiceTTS1: {
		ID:               GrokVoiceTTS1,
		Name:             "OpenRouter – Grok Voice TTS 1.0",
		Provider:         "openrouter",
		APIModel:         "x-ai/grok-voice-tts-1.0",
		CostPer1MChars:   15,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	Aura2: {
		ID:               Aura2,
		Name:             "OpenRouter – Deepgram Aura-2",
		Provider:         "openrouter",
		APIModel:         "deepgram/aura-2",
		CostPer1MChars:   30,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	Gemini31FlashTTS: {
		ID:               Gemini31FlashTTS,
		Name:             "OpenRouter – Gemini 3.1 Flash TTS Preview",
		Provider:         "openrouter",
		APIModel:         "google/gemini-3.1-flash-tts-preview",
		CostPer1MChars:   1,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	QwenAudio3TTSFlash: {
		ID:               QwenAudio3TTSFlash,
		Name:             "OpenRouter – Qwen-Audio-3.0-TTS Flash",
		Provider:         "openrouter",
		APIModel:         "qwen/qwen-audio-3.0-tts-flash",
		CostPer1MChars:   15,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	QwenAudio3TTSPlus: {
		ID:               QwenAudio3TTSPlus,
		Name:             "OpenRouter – Qwen-Audio-3.0-TTS Plus",
		Provider:         "openrouter",
		APIModel:         "qwen/qwen-audio-3.0-tts-plus",
		CostPer1MChars:   20,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	FishAudioS1: {
		ID:               FishAudioS1,
		Name:             "OpenRouter – Fish Audio S1",
		Provider:         "openrouter",
		APIModel:         "fish-audio/s1",
		CostPer1MChars:   15,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	FishAudioS2Pro: {
		ID:               FishAudioS2Pro,
		Name:             "OpenRouter – Fish Audio S2 Pro",
		Provider:         "openrouter",
		APIModel:         "fish-audio/s2-pro",
		CostPer1MChars:   15,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	FishAudioS21Pro: {
		ID:               FishAudioS21Pro,
		Name:             "OpenRouter – Fish Audio S2.1 Pro",
		Provider:         "openrouter",
		APIModel:         "fish-audio/s2.1-pro",
		CostPer1MChars:   15,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	MiniMaxSpeech28HD: {
		ID:               MiniMaxSpeech28HD,
		Name:             "OpenRouter – MiniMax Speech 2.8 HD",
		Provider:         "openrouter",
		APIModel:         "minimax/speech-2.8-hd",
		CostPer1MChars:   100,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	MiniMaxSpeech28Turbo: {
		ID:               MiniMaxSpeech28Turbo,
		Name:             "OpenRouter – MiniMax Speech 2.8 Turbo",
		Provider:         "openrouter",
		APIModel:         "minimax/speech-2.8-turbo",
		CostPer1MChars:   60,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	ZonosTransformer: {
		ID:               ZonosTransformer,
		Name:             "OpenRouter – Zonos v0.1 Transformer",
		Provider:         "openrouter",
		APIModel:         "zyphra/zonos-v0.1-transformer",
		CostPer1MChars:   7,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	ZonosHybrid: {
		ID:               ZonosHybrid,
		Name:             "OpenRouter – Zonos v0.1 Hybrid",
		Provider:         "openrouter",
		APIModel:         "zyphra/zonos-v0.1-hybrid",
		CostPer1MChars:   7,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	Orpheus3B: {
		ID:               Orpheus3B,
		Name:             "OpenRouter – Orpheus 3B",
		Provider:         "openrouter",
		APIModel:         "canopylabs/orpheus-3b-0.1-ft",
		CostPer1MChars:   7,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	CSM1B: {
		ID:               CSM1B,
		Name:             "OpenRouter – Sesame CSM 1B",
		Provider:         "openrouter",
		APIModel:         "sesame/csm-1b",
		CostPer1MChars:   7,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
	Kokoro82M: {
		ID:               Kokoro82M,
		Name:             "OpenRouter – Kokoro 82M",
		Provider:         "openrouter",
		APIModel:         "hexgrad/kokoro-82m",
		CostPer1MChars:   0.62,
		SupportedFormats: []string{"mp3", "pcm"},
		DefaultFormat:    "pcm",
	},
}
