// Package stt provides a unified interface for speech-to-text conversion using various AI providers.
//
// This package defines the [SpeechToText] interface and the data types that flow through it.
// Concrete vendor implementations live in subpackages (stt/openai, stt/deepgram, stt/google,
// stt/assemblyai, stt/elevenlabs); each subpackage exports its own NewSpeechToText constructor
// that returns a tracing-wrapped client implementing the interface.
//
// Example usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/stt"
//		sttopenai "github.com/joakimcarlsson/ai/stt/openai"
//	)
//
//	client := sttopenai.NewSpeechToText(
//		sttopenai.WithAPIKey("your-api-key"),
//		sttopenai.WithModel(model.OpenAITranscriptionModels[model.Whisper1]),
//	)
//
//	audioData, _ := os.ReadFile("audio.mp3")
//	response, err := client.Transcribe(ctx, audioData, stt.WithLanguage("en"))
package stt

import (
	"context"
	"errors"
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

// StreamResult is one event emitted by [SpeechToText.StreamTranscribe]. Interim results
// have IsFinal=false; the settled transcript is emitted with IsFinal=true. Errors are
// sent as a final StreamResult{Error: ...} value before the channel closes.
type StreamResult struct {
	Text       string
	Confidence float64
	IsFinal    bool
	WordCount  int
	Words      []Word
	Error      error
}

// ErrStreamingNotSupported is returned by [SpeechToText.StreamTranscribe] when the
// underlying provider only supports request/response transcription. Detect ahead of
// time via [SpeechToText.SupportsStreaming].
var ErrStreamingNotSupported = errors.New(
	"stt: streaming not supported by this provider",
)

// SpeechToText provides methods for converting audio to text using various AI providers.
type SpeechToText interface {
	// Transcribe converts audio to text in the same language as the audio.
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
	// session. Defaults to 16000 when supported by the provider.
	SampleRate int
	// Channels is the channel count of audio fed into a streaming session.
	// Defaults to 1 when supported by the provider.
	Channels int
}

// Option customizes a single Transcribe, Translate, or StreamTranscribe call.
type Option func(*Options)

// WithLanguage sets the expected language of the audio (ISO-639-1).
func WithLanguage(language string) Option {
	return func(o *Options) {
		o.Language = language
	}
}

// WithPrompt provides optional text to guide the model's style or continue a previous audio segment.
func WithPrompt(prompt string) Option {
	return func(o *Options) {
		o.Prompt = prompt
	}
}

// WithResponseFormat sets the output format (json, text, srt, verbose_json, vtt, or diarized_json).
func WithResponseFormat(format string) Option {
	return func(o *Options) {
		o.ResponseFormat = format
	}
}

// WithTemperature sets the sampling temperature between 0 and 1 for output randomness.
func WithTemperature(temperature float64) Option {
	return func(o *Options) {
		o.Temperature = &temperature
	}
}

// WithTimestampGranularities specifies timestamp levels to include (word, segment).
func WithTimestampGranularities(granularities ...string) Option {
	return func(o *Options) {
		o.TimestampGranularities = granularities
	}
}

// WithKnownSpeakers provides speaker names and reference audio samples for diarization.
func WithKnownSpeakers(names []string, references []string) Option {
	return func(o *Options) {
		o.KnownSpeakerNames = names
		o.KnownSpeakerReferences = references
	}
}

// WithFilename specifies the audio filename for format detection (e.g., "audio.mp3").
func WithFilename(filename string) Option {
	return func(o *Options) {
		o.Filename = filename
	}
}

// WithSampleRate declares the PCM sample rate (Hz) of audio fed into a streaming session.
// Streaming-only.
func WithSampleRate(hz int) Option {
	return func(o *Options) {
		o.SampleRate = hz
	}
}

// WithChannels declares the channel count of audio fed into a streaming session.
// Streaming-only.
func WithChannels(n int) Option {
	return func(o *Options) {
		o.Channels = n
	}
}

// TracingAttrs are construction-time attributes vendor packages forward to the
// [WithTracing] wrapper so they appear on every span produced for the wrapped
// client.
type TracingAttrs struct {
	Language string
}

// WithTracing wraps a SpeechToText client so every call records OpenTelemetry spans
// and metrics. The attrs are recorded as construction-time span attributes.
func WithTracing(inner SpeechToText, attrs TracingAttrs) SpeechToText {
	return &tracingClient{inner: inner, attrs: attrs}
}

type tracingClient struct {
	inner SpeechToText
	attrs TracingAttrs
}

func (t *tracingClient) Model() model.TranscriptionModel {
	return t.inner.Model()
}

func (t *tracingClient) SupportsStreaming() bool {
	return t.inner.SupportsStreaming()
}

func (t *tracingClient) spanAttrs() []tracing.Attr {
	var attrs []tracing.Attr
	if t.attrs.Language != "" {
		attrs = append(attrs, tracing.AttrRequestLanguage.String(t.attrs.Language))
	}
	return attrs
}

func (t *tracingClient) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartTranscribeSpan(
		ctx,
		m.APIModel,
		string(m.Provider),
		"transcribe",
		t.spanAttrs()...,
	)
	defer span.End()

	resp, err := t.inner.Transcribe(ctx, audioFile, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "transcribe",
			m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
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
		ctx, "transcribe",
		m.APIModel, string(m.Provider),
		time.Since(start),
		resp.Usage.InputTokens, resp.Usage.OutputTokens, nil,
	)
	return resp, nil
}

func (t *tracingClient) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartTranscribeSpan(
		ctx,
		m.APIModel,
		string(m.Provider),
		"translate",
		t.spanAttrs()...,
	)
	defer span.End()

	resp, err := t.inner.Translate(ctx, audioFile, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "translate",
			m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
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
		ctx, "translate",
		m.APIModel, string(m.Provider),
		time.Since(start),
		resp.Usage.InputTokens, resp.Usage.OutputTokens, nil,
	)
	return resp, nil
}

func (t *tracingClient) StreamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...Option,
) (<-chan StreamResult, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartTranscribeSpan(
		ctx,
		m.APIModel,
		string(m.Provider),
		"stream_transcribe",
		t.spanAttrs()...,
	)

	innerCh, err := t.inner.StreamTranscribe(ctx, audio, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "stream_transcribe",
			m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
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
			ctx, "stream_transcribe",
			m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, nil,
		)
	}()
	return outCh, nil
}
