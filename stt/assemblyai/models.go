package assemblyai

import (
	"github.com/joakimcarlsson/ai/stt"
)

// AssemblyAI transcription model IDs.
const (
	Best                      string = "best"
	Nano                      string = "nano"
	Universal3Pro             string = "universal-3-pro"
	Universal2                string = "universal-2"
	UniversalStreamingEnglish string = "universal-streaming-english"
	UniversalStreamingMulti   string = "universal-streaming-multilingual"
	WhisperRT                 string = "whisper-rt"
	AlphaEnglish              string = "alpha-english"
	U3RTPro                   string = "u3-rt-pro"
	U3RTAgent                 string = "u3-rt-agent"
)

// Models maps AssemblyAI model IDs to their
// configurations.
//
// Streaming model entries cover the v3 Universal Streaming endpoint
// (wss://streaming.assemblyai.com/v3/ws). Streaming pricing per AssemblyAI's
// public pricing page: Universal-Streaming $0.15/hr, Whisper-RT $0.30/hr,
// Universal-3 Pro Streaming $0.45/hr (CostPer1MIn here is the per-minute
// equivalent). Streaming endpoints accept pcm_s16le and pcm_mulaw at
// configurable sample rates with audio chunks of 50–1000 ms.
//
// Pricing source: https://www.assemblyai.com/pricing.
// Fetched: 2026-07-26.
var Models = map[string]stt.TranscriptionModel{
	Universal3Pro: {
		ID:            Universal3Pro,
		Name:          "AssemblyAI Universal-3.5 Pro",
		Provider:      "assemblyai",
		APIModel:      "universal-3-pro",
		Currency:      "USD",
		CostPer1MIn:   0.0035,
		MaxFileSizeMB: 5000,
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
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"vtt",
		},
	},
	Universal2: {
		ID:            Universal2,
		Name:          "AssemblyAI Universal-2",
		Provider:      "assemblyai",
		APIModel:      "universal-2",
		Currency:      "USD",
		CostPer1MIn:   0.0025,
		MaxFileSizeMB: 5000,
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
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"vtt",
		},
	},
	Best: {
		ID:            Best,
		Name:          "AssemblyAI Best",
		Provider:      "assemblyai",
		APIModel:      "best",
		Currency:      "USD",
		CostPer1MIn:   0.0035,
		MaxFileSizeMB: 5000,
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
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"vtt",
		},
	},
	Nano: {
		ID:            Nano,
		Name:          "AssemblyAI Nano",
		Provider:      "assemblyai",
		APIModel:      "nano",
		Currency:      "USD",
		CostPer1MIn:   0.0025,
		MaxFileSizeMB: 5000,
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
		SupportedResponseFormats: []string{
			"json",
			"text",
			"srt",
			"vtt",
		},
	},
	UniversalStreamingEnglish: {
		ID:                       UniversalStreamingEnglish,
		Name:                     "AssemblyAI Universal-Streaming English",
		Provider:                 "assemblyai",
		APIModel:                 "universal-streaming-english",
		Currency:                 "USD",
		CostPer1MIn:              0.0025,
		SupportedFormats:         []string{"pcm_s16le", "pcm_mulaw"},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	UniversalStreamingMulti: {
		ID:                       UniversalStreamingMulti,
		Name:                     "AssemblyAI Universal-Streaming Multilingual",
		Provider:                 "assemblyai",
		APIModel:                 "universal-streaming-multilingual",
		Currency:                 "USD",
		CostPer1MIn:              0.0025,
		SupportedFormats:         []string{"pcm_s16le", "pcm_mulaw"},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	WhisperRT: {
		ID:                       WhisperRT,
		Name:                     "AssemblyAI Whisper Realtime",
		Provider:                 "assemblyai",
		APIModel:                 "whisper-rt",
		Currency:                 "USD",
		CostPer1MIn:              0.005,
		SupportedFormats:         []string{"pcm_s16le", "pcm_mulaw"},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	AlphaEnglish: {
		ID:                       AlphaEnglish,
		Name:                     "AssemblyAI Alpha English",
		Provider:                 "assemblyai",
		APIModel:                 "alpha-english",
		Currency:                 "USD",
		CostPer1MIn:              0.0025,
		SupportedFormats:         []string{"pcm_s16le", "pcm_mulaw"},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	U3RTPro: {
		ID:                       U3RTPro,
		Name:                     "AssemblyAI Universal-3 Realtime Pro",
		Provider:                 "assemblyai",
		APIModel:                 "u3-rt-pro",
		Currency:                 "USD",
		CostPer1MIn:              0.0075,
		SupportedFormats:         []string{"pcm_s16le", "pcm_mulaw"},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
	U3RTAgent: {
		ID:                       U3RTAgent,
		Name:                     "AssemblyAI Universal-3 Realtime Agent",
		Provider:                 "assemblyai",
		APIModel:                 "u3-rt-agent",
		Currency:                 "USD",
		CostPer1MIn:              0.0075,
		SupportedFormats:         []string{"pcm_s16le", "pcm_mulaw"},
		SupportsTimestamps:       true,
		SupportsWordTimestamps:   true,
		SupportsStreaming:        true,
		SupportedResponseFormats: []string{"json"},
	},
}
