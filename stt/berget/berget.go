// Package berget provides a Berget AI implementation of the [stt.SpeechToText]
// interface.
//
// Berget's transcription endpoint is OpenAI-compatible (Whisper), so this wraps
// [stt/openai] pinned to Berget's base URL (https://api.berget.ai/v1). See
// [github.com/joakimcarlsson/ai/model] for the catalog
// (BergetTranscriptionModels) and pricing.
package berget

import (
	sttopenai "github.com/joakimcarlsson/ai/stt/openai"

	"github.com/joakimcarlsson/ai/stt"
)

// DefaultBaseURL is the canonical Berget AI OpenAI-compatible API endpoint.
const DefaultBaseURL = "https://api.berget.ai/v1"

// Option re-exports [sttopenai.Option] for caller convenience.
type Option = sttopenai.Option

// NewSpeechToText constructs a Berget AI speech-to-text client.
//
// [sttopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override.
func NewSpeechToText(opts ...Option) stt.SpeechToText {
	return sttopenai.NewSpeechToText(
		append([]Option{sttopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}
