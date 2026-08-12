package openai

import (
	"github.com/joakimcarlsson/ai/stt"
)

// OpenAI speech-to-text model IDs.
const (
	Whisper1                string = "whisper-1"
	GPT4oTranscribe         string = "gpt-4o-transcribe"
	GPT4oMiniTranscribe     string = "gpt-4o-mini-transcribe"
	GPT4oMiniTranscribe2025 string = "gpt-4o-mini-transcribe-2025-12-15"
	GPT4oTranscribeDiarize  string = "gpt-4o-transcribe-diarize"
	GPTRealtimeWhisper      string = "gpt-realtime-whisper"
	GPTRealtimeTranslate    string = "gpt-realtime-translate"
)

// Models contains configuration for OpenAI speech-to-text
// models.
//
// Pricing source: https://developers.openai.com/api/docs/pricing.
// Fetched: 2026-07-26.
var Models = map[string]stt.TranscriptionModel{
	Whisper1: {
		ID:            Whisper1,
		Name:          "Whisper v2",
		Provider:      "openai",
		APIModel:      "whisper-1",
		Currency:      "USD",
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
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsTranslation:    true,
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"verbose_json",
			"vtt",
		},
	},
	GPT4oTranscribe: {
		ID:            GPT4oTranscribe,
		Name:          "GPT-4o Transcribe",
		Provider:      "openai",
		APIModel:      "gpt-4o-transcribe",
		Currency:      "USD",
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
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	GPT4oMiniTranscribe: {
		ID:            GPT4oMiniTranscribe,
		Name:          "GPT-4o Mini Transcribe",
		Provider:      "openai",
		APIModel:      "gpt-4o-mini-transcribe",
		Currency:      "USD",
		CostPer1MIn:   0.003,
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
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	GPT4oMiniTranscribe2025: {
		ID:            GPT4oMiniTranscribe2025,
		Name:          "GPT-4o Mini Transcribe 2025-12-15",
		Provider:      "openai",
		APIModel:      "gpt-4o-mini-transcribe-2025-12-15",
		Currency:      "USD",
		CostPer1MIn:   0.003,
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
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	GPT4oTranscribeDiarize: {
		ID:            GPT4oTranscribeDiarize,
		Name:          "GPT-4o Transcribe Diarize",
		Provider:      "openai",
		APIModel:      "gpt-4o-transcribe-diarize",
		Currency:      "USD",
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
		SupportsDiarization:      true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json", "text", "diarized_json"},
	},
	GPTRealtimeWhisper: {
		ID:            GPTRealtimeWhisper,
		Name:          "GPT Realtime Whisper",
		Provider:      "openai",
		APIModel:      "gpt-realtime-whisper",
		Currency:      "USD",
		CostPer1MIn:   0.017,
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
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	GPTRealtimeTranslate: {
		ID:            GPTRealtimeTranslate,
		Name:          "GPT Realtime Translate",
		Provider:      "openai",
		APIModel:      "gpt-realtime-translate",
		Currency:      "USD",
		CostPer1MIn:   0.034,
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
		SupportsTranslation:      true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
}
