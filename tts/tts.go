// Package tts provides a unified interface for generating audio from text using
// various AI providers.
//
// This package defines the [Generation] interface and the data types that flow
// through it. Concrete vendor implementations live in subpackages (tts/openai,
// tts/elevenlabs, tts/google, tts/azure, tts/deepgram); each subpackage exports
// its own NewGeneration constructor that returns a tracing-wrapped client
// implementing the interface.
//
// Some vendors also implement [ForcedAlignmentProvider] (currently only ElevenLabs).
// Type-assert the constructor's return value to detect support:
//
//	client := elevenlabs.NewGeneration(...)
//	if fap, ok := client.(tts.ForcedAlignmentProvider); ok {
//		fap.GenerateForcedAlignment(ctx, audio, transcript)
//	}
//
// The [WithTracing] wrapper preserves [ForcedAlignmentProvider] when the inner
// client implements it, so the type assertion above succeeds even though the
// returned value is wrapped for tracing.
package tts

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tracing"
)

// Usage tracks the resource consumption for audio generation operations.
type Usage struct {
	// Characters is the number of characters processed.
	Characters int64
}

// AlignmentData contains character-level timing information for generated audio.
type AlignmentData struct {
	Characters                 []string
	CharacterStartTimesSeconds []float64
	CharacterEndTimesSeconds   []float64
}

// CharAlignment represents timing information for a single character.
type CharAlignment struct {
	Text  string
	Start float64
	End   float64
}

// WordAlignment represents timing information for a single word.
type WordAlignment struct {
	Text  string
	Start float64
	End   float64
	Loss  float64
}

// ForcedAlignmentData contains the response from forced alignment operations.
type ForcedAlignmentData struct {
	Characters []CharAlignment
	Words      []WordAlignment
	Loss       float64
}

// Response contains the generated audio and metadata from an audio generation request.
type Response struct {
	// AudioData contains the audio file data as bytes.
	AudioData []byte
	// ContentType specifies the MIME type of the audio data.
	ContentType string
	// Usage tracks resource consumption for this request.
	Usage Usage
	// Model identifies which audio generation model was used.
	Model string
	// Alignment contains character-level timing information aligned to the original input text.
	Alignment *AlignmentData
	// NormalizedAlignment contains character-level timing information aligned to normalized text.
	NormalizedAlignment *AlignmentData
}

// Chunk represents a piece of streaming audio data.
type Chunk struct {
	// Data contains the audio chunk bytes.
	Data []byte
	// Error contains any error that occurred during streaming.
	Error error
	// Done indicates if this is the final chunk.
	Done bool
	// Alignment contains character-level timing information for this chunk (if alignment is enabled).
	Alignment *AlignmentData
	// NormalizedAlignment contains normalized character-level timing information for this chunk.
	NormalizedAlignment *AlignmentData
}

// Voice represents an available voice for audio generation.
type Voice struct {
	VoiceID     string
	Name        string
	Category    string
	Description string
	PreviewURL  string
	Labels      map[string]string
}

// Generation defines the interface for generating audio from text.
type Generation interface {
	// GenerateAudio creates audio from text and returns the complete audio data.
	GenerateAudio(
		ctx context.Context,
		text string,
		options ...GenerationOption,
	) (*Response, error)

	// StreamAudio creates audio from text and returns a channel of audio chunks.
	StreamAudio(
		ctx context.Context,
		text string,
		options ...GenerationOption,
	) (<-chan Chunk, error)

	// ListVoices retrieves the list of available voices from the provider.
	ListVoices(ctx context.Context) ([]Voice, error)

	// Model returns the audio generation model configuration being used.
	Model() model.AudioModel
}

// ForcedAlignmentProvider is an optional sub-interface for providers that support
// forced alignment (matching existing audio with a transcript to produce timing data).
// Vendors that support it implement [GenerateForcedAlignment] on their concrete client
// type; consumers detect support via type assertion against the [Generation] returned
// from a vendor's NewGeneration constructor.
type ForcedAlignmentProvider interface {
	GenerateForcedAlignment(
		ctx context.Context,
		audioFile []byte,
		transcript string,
	) (*ForcedAlignmentData, error)
}

// GenerationOptions contains parameters for customizing audio generation requests.
type GenerationOptions struct {
	OutputFormat             string
	Stability                *float64
	SimilarityBoost          *float64
	Style                    *float64
	SpeakerBoost             *bool
	OptimizeStreamingLatency *int
	EnableAlignment          bool
}

// GenerationOption configures GenerationOptions.
type GenerationOption func(*GenerationOptions)

// WithOutputFormat sets the audio format for the generated audio.
func WithOutputFormat(format string) GenerationOption {
	return func(o *GenerationOptions) { o.OutputFormat = format }
}

// WithStability sets the voice stability (0.0 to 1.0).
func WithStability(stability float64) GenerationOption {
	return func(o *GenerationOptions) { o.Stability = &stability }
}

// WithSimilarityBoost sets how much the voice should match the original (0.0 to 1.0).
func WithSimilarityBoost(boost float64) GenerationOption {
	return func(o *GenerationOptions) { o.SimilarityBoost = &boost }
}

// WithStyle sets the style exaggeration (0.0 to 1.0).
func WithStyle(style float64) GenerationOption {
	return func(o *GenerationOptions) { o.Style = &style }
}

// WithSpeakerBoost enables or disables speaker boost for enhanced similarity.
func WithSpeakerBoost(enabled bool) GenerationOption {
	return func(o *GenerationOptions) { o.SpeakerBoost = &enabled }
}

// WithOptimizeStreamingLatency sets the streaming latency optimization level (0-4).
func WithOptimizeStreamingLatency(level int) GenerationOption {
	return func(o *GenerationOptions) { o.OptimizeStreamingLatency = &level }
}

// WithAlignmentEnabled enables character-level timing data in the response.
func WithAlignmentEnabled(enabled bool) GenerationOption {
	return func(o *GenerationOptions) { o.EnableAlignment = enabled }
}

// WithTracing wraps a Generation client so every call records OpenTelemetry spans
// and metrics. If the inner client also implements [ForcedAlignmentProvider], the
// returned wrapper does too — type assertion on the wrapper succeeds and the call
// is traced and forwarded to the inner client.
func WithTracing(inner Generation) Generation {
	base := &tracingGeneration{inner: inner}
	if fap, ok := inner.(ForcedAlignmentProvider); ok {
		return &tracingGenerationWithForcedAlignment{
			tracingGeneration: base,
			fap:               fap,
		}
	}
	return base
}

type tracingGeneration struct {
	inner Generation
}

func (t *tracingGeneration) Model() model.AudioModel {
	return t.inner.Model()
}

func (t *tracingGeneration) ListVoices(ctx context.Context) ([]Voice, error) {
	return t.inner.ListVoices(ctx)
}

func (t *tracingGeneration) GenerateAudio(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (*Response, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartAudioSpan(ctx, m.APIModel, string(m.Provider))
	defer span.End()
	span.SetAttributes(tracing.AttrInputCount.Int(len(text)))

	resp, err := t.inner.GenerateAudio(ctx, text, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "generate_audio", m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageCharacters.Int64(int64(resp.Usage.Characters)),
	)
	tracing.RecordMetrics(
		ctx, "generate_audio", m.APIModel, string(m.Provider),
		time.Since(start), 0, 0, nil,
	)
	return resp, nil
}

func (t *tracingGeneration) StreamAudio(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (<-chan Chunk, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartAudioSpan(ctx, m.APIModel, string(m.Provider))
	span.SetAttributes(tracing.AttrInputCount.Int(len(text)))

	innerCh, err := t.inner.StreamAudio(ctx, text, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "generate_audio", m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
		)
		span.End()
		return nil, err
	}

	outCh := make(chan Chunk)
	go func() {
		defer close(outCh)
		defer span.End()
		for chunk := range innerCh {
			if chunk.Error != nil {
				tracing.SetError(span, chunk.Error)
			}
			outCh <- chunk
		}
		tracing.RecordMetrics(
			ctx, "generate_audio", m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, nil,
		)
	}()
	return outCh, nil
}

// tracingGenerationWithForcedAlignment is the tracing wrapper used when the inner
// Generation client also implements ForcedAlignmentProvider. The type-assertion
// `c.(tts.ForcedAlignmentProvider)` against the wrapper returned from NewGeneration
// succeeds for vendors that support forced alignment (currently only ElevenLabs).
type tracingGenerationWithForcedAlignment struct {
	*tracingGeneration
	fap ForcedAlignmentProvider
}

func (t *tracingGenerationWithForcedAlignment) GenerateForcedAlignment(
	ctx context.Context,
	audioFile []byte,
	transcript string,
) (*ForcedAlignmentData, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartAudioSpan(ctx, m.APIModel, string(m.Provider))
	defer span.End()
	span.SetAttributes(tracing.AttrInputCount.Int(len(transcript)))

	resp, err := t.fap.GenerateForcedAlignment(ctx, audioFile, transcript)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, "forced_alignment", m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, err,
		)
		return nil, err
	}
	tracing.RecordMetrics(
		ctx, "forced_alignment", m.APIModel, string(m.Provider),
		time.Since(start), 0, 0, nil,
	)
	return resp, nil
}
