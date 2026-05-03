// Package stt provides a unified interface for speech-to-text conversion using various AI providers.
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
//	client, err := stt.NewSpeechToText(
//		model.ProviderOpenAI,
//		stt.WithAPIKey("your-api-key"),
//		stt.WithModel(model.OpenAITranscriptionModels[model.Whisper1]),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	audioData, _ := os.ReadFile("tts.mp3")
//	response, err := client.Transcribe(ctx, audioData,
//		stt.WithLanguage("en"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println(response.Text)
package stt

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tracing"
)

// Usage tracks resource consumption for transcription operations.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	AudioTokens  int64
	TextTokens   int64
	DurationSec  float64
}

// Segment represents a segment of transcribed audio with timing and metadata.
type Segment struct {
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

// Word represents a single word with its timing information.
type Word struct {
	Word  string
	Start float64
	End   float64
}

// Response contains the transcription result with optional segments, words, and usage data.
type Response struct {
	Text     string
	Language string
	Duration float64
	Segments []Segment
	Words    []Word
	Usage    Usage
	Model    string
}

// SpeechToText provides methods for converting audio to text using various AI providers.
type SpeechToText interface {
	// Transcribe converts audio to text in the same language as the tts.

	Transcribe(
		ctx context.Context,
		audioFile []byte,
		options ...Option,
	) (*Response, error)

	// Translate converts audio to English text regardless of the source language.
	Translate(
		ctx context.Context,
		audioFile []byte,
		options ...Option,
	) (*Response, error)

	// StreamTranscribe opens a streaming transcription session. Returns
	// ErrStreamingNotSupported when the underlying provider does not support
	// streaming — check ahead with SupportsStreaming.
	StreamTranscribe(
		ctx context.Context,
		audio <-chan []byte,
		options ...Option,
	) (<-chan StreamResult, error)

	// SupportsStreaming reports whether this client can serve StreamTranscribe.
	SupportsStreaming() bool

	// Model returns the transcription model configuration being used.
	Model() model.TranscriptionModel
}

type transcriptionClientOptions struct {
	apiKey  string
	model   model.TranscriptionModel
	timeout *time.Duration

	streamSampleRate int
	streamChannels   int

	openaiOptions         []OpenAIOption
	deepgramOptions       []DeepgramOption
	googleCloudSTTOptions []GoogleCloudSTTOption
	assemblyAIOptions     []AssemblyAIOption
	elevenLabsOptions     []ElevenLabsOption
}

// ClientOption configures a speech-to-text client.
type ClientOption func(*transcriptionClientOptions)

// SpeechToTextClient is the internal interface implemented by provider-specific transcription clients.
type SpeechToTextClient interface {
	transcribe(
		ctx context.Context,
		audioFile []byte,
		options ...Option,
	) (*Response, error)
	translate(
		ctx context.Context,
		audioFile []byte,
		options ...Option,
	) (*Response, error)
}

type baseSpeechToText[C SpeechToTextClient] struct {
	options   transcriptionClientOptions
	client    C
	streaming streamingSpeechToTextClient
}

// NewSpeechToText creates a new speech-to-text client for the specified provider.
func NewSpeechToText(
	provider model.Provider,
	opts ...ClientOption,
) (SpeechToText, error) {
	clientOptions := transcriptionClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderOpenAI:
		c := newOpenAIClient(clientOptions)
		return &baseSpeechToText[OpenAIClient]{
			options:   clientOptions,
			client:    c,
			streaming: resolveStreaming(c),
		}, nil
	case model.ProviderDeepgram:
		c := newDeepgramClient(clientOptions)
		return &baseSpeechToText[DeepgramClient]{
			options:   clientOptions,
			client:    c,
			streaming: resolveStreaming(c),
		}, nil
	case model.ProviderGoogleCloud:
		c := newGoogleCloudClient(clientOptions)
		return &baseSpeechToText[GoogleCloudClient]{
			options:   clientOptions,
			client:    c,
			streaming: resolveStreaming(c),
		}, nil
	case model.ProviderAssemblyAI:
		c := newAssemblyAIClient(clientOptions)
		return &baseSpeechToText[AssemblyAIClient]{
			options:   clientOptions,
			client:    c,
			streaming: resolveStreaming(c),
		}, nil
	case model.ProviderElevenLabs:
		c := newElevenLabsClient(clientOptions)
		return &baseSpeechToText[ElevenLabsClient]{
			options:   clientOptions,
			client:    c,
			streaming: resolveStreaming(c),
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
	options ...Option,
) (*Response, error) {
	start := time.Now()
	ctx, span := tracing.StartTranscribeSpan(
		ctx,
		s.options.model.APIModel,
		string(s.options.model.Provider),
		"transcribe",
	)
	defer span.End()

	resp, err := s.client.transcribe(ctx, audioFile, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx,
			"transcribe",
			s.options.model.APIModel,
			string(s.options.model.Provider),
			time.Since(start),
			0,
			0,
			err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageInputTokens.Int64(resp.Usage.InputTokens),
		tracing.AttrUsageOutputTokens.Int64(resp.Usage.OutputTokens),
		tracing.AttrDurationSec.Float64(resp.Duration),
		tracing.AttrLanguage.String(resp.Language),
	)
	tracing.RecordMetrics(
		ctx,
		"transcribe",
		s.options.model.APIModel,
		string(s.options.model.Provider),
		time.Since(start),
		resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
		nil,
	)
	return resp, nil
}

func (s *baseSpeechToText[C]) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	start := time.Now()
	ctx, span := tracing.StartTranscribeSpan(
		ctx,
		s.options.model.APIModel,
		string(s.options.model.Provider),
		"translate",
	)
	defer span.End()

	resp, err := s.client.translate(ctx, audioFile, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx,
			"translate",
			s.options.model.APIModel,
			string(s.options.model.Provider),
			time.Since(start),
			0,
			0,
			err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageInputTokens.Int64(resp.Usage.InputTokens),
		tracing.AttrUsageOutputTokens.Int64(resp.Usage.OutputTokens),
		tracing.AttrDurationSec.Float64(resp.Duration),
		tracing.AttrLanguage.String(resp.Language),
	)
	tracing.RecordMetrics(
		ctx,
		"translate",
		s.options.model.APIModel,
		string(s.options.model.Provider),
		time.Since(start),
		resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
		nil,
	)
	return resp, nil
}

func (s *baseSpeechToText[C]) Model() model.TranscriptionModel {
	return s.options.model
}

// SupportsStreaming reports whether this client can serve StreamTranscribe.
func (s *baseSpeechToText[C]) SupportsStreaming() bool {
	return s.streaming != nil
}

// StreamTranscribe opens a streaming session against the provider.
func (s *baseSpeechToText[C]) StreamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...Option,
) (<-chan StreamResult, error) {
	if s.streaming == nil {
		return nil, ErrStreamingNotSupported
	}
	start := time.Now()
	ctx, span := tracing.StartTranscribeSpan(
		ctx,
		s.options.model.APIModel,
		string(s.options.model.Provider),
		"stream_transcribe",
	)

	defaults := s.streamDefaults()
	innerCh, err := s.streaming.streamTranscribe(
		ctx,
		audio,
		append(defaults, options...)...,
	)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx,
			"stream_transcribe",
			s.options.model.APIModel,
			string(s.options.model.Provider),
			time.Since(start),
			0,
			0,
			err,
		)
		span.End()
		return nil, err
	}

	outCh := make(chan StreamResult)
	go func() {
		defer close(outCh)
		defer span.End()
		for r := range innerCh {
			if r.Error != nil {
				tracing.SetError(span, r.Error)
			}
			outCh <- r
		}
		tracing.RecordMetrics(
			ctx,
			"stream_transcribe",
			s.options.model.APIModel,
			string(s.options.model.Provider),
			time.Since(start),
			0,
			0,
			nil,
		)
	}()
	return outCh, nil
}

// resolveStreaming returns the streaming sub-interface implementation if the
// concrete client supports it, otherwise nil. Cached at construction.
func resolveStreaming[C SpeechToTextClient](c C) streamingSpeechToTextClient {
	if s, ok := any(c).(streamingSpeechToTextClient); ok {
		return s
	}
	return nil
}

func (s *baseSpeechToText[C]) streamDefaults() []Option {
	var defaults []Option
	if s.options.streamSampleRate != 0 {
		rate := s.options.streamSampleRate
		defaults = append(defaults, func(o *Options) { o.SampleRate = rate })
	}
	if s.options.streamChannels != 0 {
		ch := s.options.streamChannels
		defaults = append(defaults, func(o *Options) { o.Channels = ch })
	}
	return defaults
}

// WithAPIKey sets the API key for authentication with the speech-to-text provider.
func WithAPIKey(apiKey string) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.apiKey = apiKey
	}
}

// WithModel specifies which transcription model to use.
func WithModel(model model.TranscriptionModel) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.model = model
	}
}

// WithTimeout sets the maximum duration to wait for transcription requests to complete.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.timeout = &timeout
	}
}

// WithStreamSampleRate declares the PCM sample rate (Hz) of the audio fed
// into streaming sessions. Set at client construction time. Default is
// 16000 when supported by the provider.
func WithStreamSampleRate(hz int) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.streamSampleRate = hz
	}
}

// WithStreamChannels declares the channel count of the audio fed into
// streaming sessions. Set at client construction time. Default is 1 when
// supported by the provider.
func WithStreamChannels(n int) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.streamChannels = n
	}
}

// WithOpenAIOptions applies OpenAI-specific configuration options.
func WithOpenAIOptions(
	openaiOptions ...OpenAIOption,
) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.openaiOptions = openaiOptions
	}
}

// WithDeepgramOptions applies Deepgram-specific configuration options.
func WithDeepgramOptions(
	deepgramOptions ...DeepgramOption,
) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.deepgramOptions = deepgramOptions
	}
}

// WithGoogleCloudSTTOptions applies Google Cloud STT-specific configuration options.
func WithGoogleCloudSTTOptions(
	gcOptions ...GoogleCloudSTTOption,
) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.googleCloudSTTOptions = gcOptions
	}
}

// WithAssemblyAIOptions applies AssemblyAI-specific configuration options.
func WithAssemblyAIOptions(
	aaiOptions ...AssemblyAIOption,
) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.assemblyAIOptions = aaiOptions
	}
}

// WithElevenLabsSTTOptions applies ElevenLabs Scribe-specific configuration options.
func WithElevenLabsSTTOptions(
	elOptions ...ElevenLabsOption,
) ClientOption {
	return func(options *transcriptionClientOptions) {
		options.elevenLabsOptions = elOptions
	}
}

// Options contains parameters for customizing transcription requests.
type Options struct {
	Language               string
	Prompt                 string
	ResponseFormat         string
	Temperature            *float64
	TimestampGranularities []string
	KnownSpeakerNames      []string
	KnownSpeakerReferences []string
	Filename               string
	// SampleRate is the PCM sample rate (Hz) of audio fed into a streaming
	// session. Set via WithStreamSampleRate. Defaults to 16000.
	SampleRate int
	// Channels is the channel count of audio fed into a streaming session.
	// Set via WithStreamChannels. Defaults to 1.
	Channels int
}

// Option customizes a single Transcribe or Translate call.
type Option func(*Options)

// WithLanguage sets the expected language of the audio (ISO-639-1).
func WithLanguage(language string) Option {
	return func(options *Options) {
		options.Language = language
	}
}

// WithPrompt provides optional text to guide the model's style or continue a previous audio segment.
func WithPrompt(prompt string) Option {
	return func(options *Options) {
		options.Prompt = prompt
	}
}

// WithResponseFormat sets the output format (json, text, srt, verbose_json, vtt, or diarized_json).
func WithResponseFormat(format string) Option {
	return func(options *Options) {
		options.ResponseFormat = format
	}
}

// WithTemperature sets the sampling temperature between 0 and 1 for output randomness.
func WithTemperature(temperature float64) Option {
	return func(options *Options) {
		options.Temperature = &temperature
	}
}

// WithTimestampGranularities specifies timestamp levels to include (word, segment).
func WithTimestampGranularities(granularities ...string) Option {
	return func(options *Options) {
		options.TimestampGranularities = granularities
	}
}

// WithKnownSpeakers provides speaker names and reference audio samples for diarization.
func WithKnownSpeakers(
	names []string,
	references []string,
) Option {
	return func(options *Options) {
		options.KnownSpeakerNames = names
		options.KnownSpeakerReferences = references
	}
}

// WithFilename specifies the audio filename for format detection (e.g., "tts.mp3").
func WithFilename(filename string) Option {
	return func(options *Options) {
		options.Filename = filename
	}
}

// OpenAIOption configures OpenAI-specific transcription client settings.
type OpenAIOption func(*openaiOptions)

type openaiOptions struct {
	baseURL  string
	language string
}

// WithOpenAIBaseURL sets a custom base URL for the OpenAI API.
func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(o *openaiOptions) {
		o.baseURL = baseURL
	}
}

// WithOpenAISTTLanguage sets the default language (ISO-639-1, e.g. "en",
// "sv") for stt. Set at client construction time. Per-call
// WithLanguage overrides this when supplied.
func WithOpenAISTTLanguage(language string) OpenAIOption {
	return func(o *openaiOptions) {
		o.language = language
	}
}
