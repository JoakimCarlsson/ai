// Package transcription provides a unified interface for speech-to-text conversion using various AI providers.
//
// This package abstracts away the differences between speech-to-text providers like OpenAI Whisper,
// providing a consistent API for transcribing audio files and translating them to English.
//
// Key features include:
//   - Multi-provider support (OpenAI Whisper with more providers coming)
//   - Audio transcription in the same language
//   - Audio translation to English
//   - Timestamp support (word and segment level)
//   - Multiple output formats (json, text, srt, vtt, verbose_json)
//   - Token and duration-based usage tracking
//
// Example usage:
//
//	client, err := transcription.NewSpeechToText(
//		model.ProviderOpenAI,
//		transcription.WithAPIKey("your-api-key"),
//		transcription.WithModel(model.OpenAITranscriptionModels[model.Whisper1]),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	audioData, _ := os.ReadFile("audio.mp3")
//	response, err := client.Transcribe(ctx, audioData,
//		transcription.WithLanguage("en"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println(response.Text)
package transcription

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// TranscriptionUsage tracks resource consumption for transcription operations.
type TranscriptionUsage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	AudioTokens  int64
	TextTokens   int64
	DurationSec  float64
}

// TranscriptionSegment represents a segment of transcribed audio with timing and metadata.
type TranscriptionSegment struct {
	ID               int
	Start            float64
	End              float64
	Text             string
	Tokens           []int
	Temperature      float64
	AvgLogprob       float64
	CompressionRatio float64
	NoSpeechProb     float64
	Speaker          string
}

// TranscriptionWord represents a single word with its timing information.
type TranscriptionWord struct {
	Word  string
	Start float64
	End   float64
}

// TranscriptionResponse contains the transcription result with optional segments, words, and usage data.
type TranscriptionResponse struct {
	Text     string
	Language string
	Duration float64
	Segments []TranscriptionSegment
	Words    []TranscriptionWord
	Usage    TranscriptionUsage
	Model    string
}

// SpeechToText provides methods for converting audio to text using various AI providers.
type SpeechToText interface {
	// Transcribe converts audio to text in the same language as the audio.

	Transcribe(
		ctx context.Context,
		audioFile []byte,
		options ...TranscriptionOption,
	) (*TranscriptionResponse, error)

	// Translate converts audio to English text regardless of the source language.
	Translate(
		ctx context.Context,
		audioFile []byte,
		options ...TranscriptionOption,
	) (*TranscriptionResponse, error)

	// Model returns the transcription model configuration being used.
	Model() model.TranscriptionModel
}

type transcriptionClientOptions struct {
	apiKey  string
	model   model.TranscriptionModel
	timeout *time.Duration

	openaiOptions []OpenAIOption
}

type TranscriptionClientOption func(*transcriptionClientOptions)

type SpeechToTextClient interface {
	transcribe(
		ctx context.Context,
		audioFile []byte,
		options ...TranscriptionOption,
	) (*TranscriptionResponse, error)
	translate(
		ctx context.Context,
		audioFile []byte,
		options ...TranscriptionOption,
	) (*TranscriptionResponse, error)
}

type baseSpeechToText[C SpeechToTextClient] struct {
	options transcriptionClientOptions
	client  C
}

// NewSpeechToText creates a new speech-to-text client for the specified provider.
func NewSpeechToText(
	provider model.ModelProvider,
	opts ...TranscriptionClientOption,
) (SpeechToText, error) {
	clientOptions := transcriptionClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderOpenAI:
		return &baseSpeechToText[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf(
		"speech-to-text provider not supported: %s",
		provider,
	)
}

func (s *baseSpeechToText[C]) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...TranscriptionOption,
) (*TranscriptionResponse, error) {
	return s.client.transcribe(ctx, audioFile, options...)
}

func (s *baseSpeechToText[C]) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...TranscriptionOption,
) (*TranscriptionResponse, error) {
	return s.client.translate(ctx, audioFile, options...)
}

func (s *baseSpeechToText[C]) Model() model.TranscriptionModel {
	return s.options.model
}

// WithAPIKey sets the API key for authentication with the speech-to-text provider.
func WithAPIKey(apiKey string) TranscriptionClientOption {
	return func(options *transcriptionClientOptions) {
		options.apiKey = apiKey
	}
}

// WithModel specifies which transcription model to use.
func WithModel(model model.TranscriptionModel) TranscriptionClientOption {
	return func(options *transcriptionClientOptions) {
		options.model = model
	}
}

// WithTimeout sets the maximum duration to wait for transcription requests to complete.
func WithTimeout(timeout time.Duration) TranscriptionClientOption {
	return func(options *transcriptionClientOptions) {
		options.timeout = &timeout
	}
}

// WithOpenAIOptions applies OpenAI-specific configuration options.
func WithOpenAIOptions(openaiOptions ...OpenAIOption) TranscriptionClientOption {
	return func(options *transcriptionClientOptions) {
		options.openaiOptions = openaiOptions
	}
}

// TranscriptionOptions contains parameters for customizing transcription requests.
type TranscriptionOptions struct {
	Language               string
	Prompt                 string
	ResponseFormat         string
	Temperature            *float64
	TimestampGranularities []string
	KnownSpeakerNames      []string
	KnownSpeakerReferences []string
	Filename               string
}

type TranscriptionOption func(*TranscriptionOptions)

func WithLanguage(language string) TranscriptionOption {
	return func(options *TranscriptionOptions) {
		options.Language = language
	}
}

// WithPrompt provides optional text to guide the model's style or continue a previous audio segment.
func WithPrompt(prompt string) TranscriptionOption {
	return func(options *TranscriptionOptions) {
		options.Prompt = prompt
	}
}

// WithResponseFormat sets the output format (json, text, srt, verbose_json, vtt, or diarized_json).
func WithResponseFormat(format string) TranscriptionOption {
	return func(options *TranscriptionOptions) {
		options.ResponseFormat = format
	}
}

// WithTemperature sets the sampling temperature between 0 and 1 for output randomness.
func WithTemperature(temperature float64) TranscriptionOption {
	return func(options *TranscriptionOptions) {
		options.Temperature = &temperature
	}
}

// WithTimestampGranularities specifies timestamp levels to include (word, segment).
func WithTimestampGranularities(granularities ...string) TranscriptionOption {
	return func(options *TranscriptionOptions) {
		options.TimestampGranularities = granularities
	}
}

// WithKnownSpeakers provides speaker names and reference audio samples for diarization.
func WithKnownSpeakers(names []string, references []string) TranscriptionOption {
	return func(options *TranscriptionOptions) {
		options.KnownSpeakerNames = names
		options.KnownSpeakerReferences = references
	}
}

// WithFilename specifies the audio filename for format detection (e.g., "audio.mp3").
func WithFilename(filename string) TranscriptionOption {
	return func(options *TranscriptionOptions) {
		options.Filename = filename
	}
}

type OpenAIOption func(*openaiOptions)

type openaiOptions struct {
}
