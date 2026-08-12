package openrouter

import (
	"github.com/joakimcarlsson/ai/stt"
)

// OpenRouter speech-to-text model IDs.
const (
	Whisper1              string = "openrouter.whisper-1"
	WhisperLargeV3        string = "openrouter.whisper-large-v3"
	WhisperLargeV3Turbo   string = "openrouter.whisper-large-v3-turbo"
	GPT4oTranscribe       string = "openrouter.gpt-4o-transcribe"
	GPT4oMiniTranscribe   string = "openrouter.gpt-4o-mini-transcribe"
	VoxtralMiniTranscribe string = "openrouter.voxtral-mini-transcribe"
	FishAudioTranscribe1  string = "openrouter.transcribe-1"
	GrokSTT1              string = "openrouter.grok-stt-1.0"
	Nova3                 string = "openrouter.nova-3"
	MAITranscribe15       string = "openrouter.mai-transcribe-1.5"
	ParakeetTDT06BV3      string = "openrouter.parakeet-tdt-0.6b-v3"
	Qwen3ASRFlash         string = "openrouter.qwen3-asr-flash-2026-02-10"
	Chirp3                string = "openrouter.chirp-3"
)

// Models maps OpenRouter speech-to-text model IDs to
// their configurations.
//
// Known-good defaults; any OpenRouter transcription model id works even without
// an entry here.
//
// Source: the OpenRouter models API filtered to transcription outputs, plus the
// speech-to-text guide. Fetched: 2026-07-31.
//
// Two OpenRouter-wide caveats are encoded here. verbose_json — and with it
// segments and word timestamps — is only accepted by the OpenAI-compatible
// upstreams (OpenAI, Groq, Together); the rest reject it with HTTP 400, so
// SupportsTimestamps is false for those. OpenRouter exposes no
// /audio/translations route at all, so SupportsTranslation is false for every
// entry regardless of what the upstream model itself can do.
//
// Rates match the upstream provider's published rate where OpenRouter passes
// it through.
// Where OpenRouter quotes a rate in a unit TranscriptionModel cannot express
// (per audio minute rather than per token) the cost fields are left zero rather
// than converted into a fabricated per-token figure.
var Models = map[string]stt.TranscriptionModel{
	Whisper1: {
		ID:            Whisper1,
		Name:          "OpenRouter – Whisper v2",
		Provider:      "openrouter",
		APIModel:      "openai/whisper-1",
		CostPer1MIn:   0.006,
		MaxFileSizeMB: 25,
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
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportedResponseFormats: []string{"json", "verbose_json"},
	},
	WhisperLargeV3: {
		ID:            WhisperLargeV3,
		Name:          "OpenRouter – Whisper Large v3",
		Provider:      "openrouter",
		APIModel:      "openai/whisper-large-v3",
		MaxFileSizeMB: 25,
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
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportedResponseFormats: []string{"json", "verbose_json"},
	},
	WhisperLargeV3Turbo: {
		ID:            WhisperLargeV3Turbo,
		Name:          "OpenRouter – Whisper Large v3 Turbo",
		Provider:      "openrouter",
		APIModel:      "openai/whisper-large-v3-turbo",
		MaxFileSizeMB: 25,
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
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportedResponseFormats: []string{"json", "verbose_json"},
	},
	GPT4oTranscribe: {
		ID:            GPT4oTranscribe,
		Name:          "OpenRouter – GPT-4o Transcribe",
		Provider:      "openrouter",
		APIModel:      "openai/gpt-4o-transcribe",
		CostPer1MIn:   2.5,
		CostPer1MOut:  10,
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	GPT4oMiniTranscribe: {
		ID:            GPT4oMiniTranscribe,
		Name:          "OpenRouter – GPT-4o Mini Transcribe",
		Provider:      "openrouter",
		APIModel:      "openai/gpt-4o-mini-transcribe",
		CostPer1MIn:   1.25,
		CostPer1MOut:  5,
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	VoxtralMiniTranscribe: {
		ID:            VoxtralMiniTranscribe,
		Name:          "OpenRouter – Voxtral Mini Transcribe",
		Provider:      "openrouter",
		APIModel:      "mistralai/voxtral-mini-transcribe",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	FishAudioTranscribe1: {
		ID:            FishAudioTranscribe1,
		Name:          "OpenRouter – Fish Audio Transcribe 1",
		Provider:      "openrouter",
		APIModel:      "fish-audio/transcribe-1",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	GrokSTT1: {
		ID:            GrokSTT1,
		Name:          "OpenRouter – Grok STT 1.0",
		Provider:      "openrouter",
		APIModel:      "x-ai/grok-stt-1.0",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	Nova3: {
		ID:            Nova3,
		Name:          "OpenRouter – Deepgram Nova-3",
		Provider:      "openrouter",
		APIModel:      "deepgram/nova-3",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	MAITranscribe15: {
		ID:            MAITranscribe15,
		Name:          "OpenRouter – MAI-Transcribe 1.5",
		Provider:      "openrouter",
		APIModel:      "microsoft/mai-transcribe-1.5",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	ParakeetTDT06BV3: {
		ID:            ParakeetTDT06BV3,
		Name:          "OpenRouter – Parakeet TDT 0.6B v3",
		Provider:      "openrouter",
		APIModel:      "nvidia/parakeet-tdt-0.6b-v3",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	Qwen3ASRFlash: {
		ID:            Qwen3ASRFlash,
		Name:          "OpenRouter – Qwen3 ASR Flash",
		Provider:      "openrouter",
		APIModel:      "qwen/qwen3-asr-flash-2026-02-10",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
	Chirp3: {
		ID:            Chirp3,
		Name:          "OpenRouter – Chirp 3",
		Provider:      "openrouter",
		APIModel:      "google/chirp-3",
		MaxFileSizeMB: 25,
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
		SupportedResponseFormats: []string{"json"},
	},
}
