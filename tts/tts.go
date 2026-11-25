// Package tts provides a unified interface for generating audio from text using
// various AI providers.
//
// This package defines the [Generation] interface and the data types that flow
// through it. Concrete vendor implementations live in subpackages (tts/openai,
// tts/elevenlabs, tts/google, tts/azure, tts/deepgram); each subpackage exports
// its own NewGeneration constructor that returns a tracing-wrapped client
// implementing the interface.
//
// Some vendors also implement the optional [ForcedAlignmentProvider],
// [StreamingTextProvider] and [DialogueProvider] interfaces. Type-assert the
// constructor's return value to detect support:
//
//	client := elevenlabs.NewGeneration(...)
//	if fap, ok := client.(tts.ForcedAlignmentProvider); ok {
//		fap.GenerateForcedAlignment(ctx, audio, transcript)
//	}
//
// The [WithTracing] wrapper preserves every optional interface the inner
// client implements, so the type assertion above succeeds even though the
// returned value is wrapped for tracing.
package tts

import (
	"context"
	"time"

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
	Model() AudioModel
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

// StreamingTextProvider is an optional sub-interface for providers that accept
// text incrementally and produce audio as the text arrives (e.g. forwarding LLM
// deltas to a TTS WebSocket). Implementations consume textIn until the channel
// is closed by the caller, then signal end-of-input to the provider.
//
// Detect support via type assertion on the [Generation] returned from a vendor's
// NewGeneration constructor. The [WithTracing] wrapper preserves this interface
// when the inner client implements it.
type StreamingTextProvider interface {
	StreamAudioFromText(
		ctx context.Context,
		textIn <-chan string,
		options ...GenerationOption,
	) (<-chan Chunk, error)
}

// DialogueTurn is one line of a multi-speaker dialogue: the text one voice
// speaks.
type DialogueTurn struct {
	// Text is what the voice says.
	Text string
	// VoiceID is the provider's ID of the voice that speaks Text.
	VoiceID string
	// NewTurn marks Text as the start of a new speaker turn when streaming.
	// A change of VoiceID also starts a new turn. Ignored by GenerateDialogue,
	// where every DialogueTurn is its own turn.
	NewTurn bool
}

// DialogueProvider is an optional sub-interface for providers that render a
// conversation between several voices as one piece of audio.
//
// Detect support via type assertion on the [Generation] returned from a
// vendor's NewGeneration constructor. The [WithTracing] wrapper preserves this
// interface when the inner client implements it.
type DialogueProvider interface {
	// GenerateDialogue renders turns, in order, as a single audio response.
	GenerateDialogue(
		ctx context.Context,
		turns []DialogueTurn,
		options ...GenerationOption,
	) (*Response, error)

	// StreamDialogue registers voiceIDs for the session, then renders turns
	// as they arrive and emits audio chunks while it does. Every VoiceID sent
	// on turns must be in voiceIDs. Closing turns ends the input; the chunk
	// channel closes once the provider has sent the last audio.
	StreamDialogue(
		ctx context.Context,
		voiceIDs []string,
		turns <-chan DialogueTurn,
		options ...GenerationOption,
	) (<-chan Chunk, error)
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
	LanguageCode             string
	Seed                     *uint32
	TextNormalization        string
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

// WithLanguageCode forces the language of the generated speech and of text
// normalization, as an ISO 639-1 code (or ISO 639-3 where none exists).
func WithLanguageCode(code string) GenerationOption {
	return func(o *GenerationOptions) { o.LanguageCode = code }
}

// WithSeed asks the provider to sample deterministically, so repeated requests
// with the same seed and parameters return the same audio where it can.
func WithSeed(seed uint32) GenerationOption {
	return func(o *GenerationOptions) { o.Seed = &seed }
}

// WithTextNormalization sets whether numbers, dates and similar text are
// spelled out before synthesis: "auto", "on" or "off".
func WithTextNormalization(mode string) GenerationOption {
	return func(o *GenerationOptions) { o.TextNormalization = mode }
}

// TracingAttrs are construction-time attributes vendor packages forward to the
// [WithTracing] wrapper so they appear on every span produced for the wrapped
// client.
type TracingAttrs struct {
	Voice        string
	OutputFormat string
	Speed        *float64
	Language     string
	SampleRate   int
}

// WithTracing wraps a Generation client so every call records OpenTelemetry
// spans and metrics. Every optional interface the inner client implements
// ([ForcedAlignmentProvider], [StreamingTextProvider], [DialogueProvider]) is
// preserved: type assertions on the wrapper succeed and the call is traced and
// forwarded to the inner client. Interfaces the inner client lacks are absent
// from the wrapper too.
func WithTracing(inner Generation, attrs TracingAttrs) Generation {
	base := &tracingGeneration{inner: inner, attrs: attrs}
	var fa *tracedForcedAlignment
	if fap, ok := inner.(ForcedAlignmentProvider); ok {
		fa = &tracedForcedAlignment{t: base, fap: fap}
	}
	var st *tracedStreamingText
	if stp, ok := inner.(StreamingTextProvider); ok {
		st = &tracedStreamingText{t: base, stp: stp}
	}
	var dl *tracedDialogue
	if dp, ok := inner.(DialogueProvider); ok {
		dl = &tracedDialogue{t: base, dp: dp}
	}

	switch {
	case fa != nil && st != nil && dl != nil:
		return struct {
			*tracingGeneration
			*tracedForcedAlignment
			*tracedStreamingText
			*tracedDialogue
		}{base, fa, st, dl}
	case fa != nil && st != nil:
		return struct {
			*tracingGeneration
			*tracedForcedAlignment
			*tracedStreamingText
		}{base, fa, st}
	case fa != nil && dl != nil:
		return struct {
			*tracingGeneration
			*tracedForcedAlignment
			*tracedDialogue
		}{base, fa, dl}
	case st != nil && dl != nil:
		return struct {
			*tracingGeneration
			*tracedStreamingText
			*tracedDialogue
		}{base, st, dl}
	case fa != nil:
		return struct {
			*tracingGeneration
			*tracedForcedAlignment
		}{base, fa}
	case st != nil:
		return struct {
			*tracingGeneration
			*tracedStreamingText
		}{base, st}
	case dl != nil:
		return struct {
			*tracingGeneration
			*tracedDialogue
		}{base, dl}
	}
	return base
}

type tracingGeneration struct {
	inner Generation
	attrs TracingAttrs
}

func (t *tracingGeneration) Model() AudioModel {
	return t.inner.Model()
}

func (t *tracingGeneration) ListVoices(ctx context.Context) ([]Voice, error) {
	return t.inner.ListVoices(ctx)
}

func (t *tracingGeneration) spanAttrs() []tracing.Attr {
	var attrs []tracing.Attr
	if t.attrs.Voice != "" {
		attrs = append(attrs, tracing.AttrRequestVoice.String(t.attrs.Voice))
	}
	if t.attrs.OutputFormat != "" {
		attrs = append(
			attrs,
			tracing.AttrRequestOutputFormat.String(t.attrs.OutputFormat),
		)
	}
	if t.attrs.Speed != nil {
		attrs = append(attrs, tracing.AttrRequestSpeed.Float64(*t.attrs.Speed))
	}
	if t.attrs.Language != "" {
		attrs = append(
			attrs,
			tracing.AttrRequestLanguage.String(t.attrs.Language),
		)
	}
	if t.attrs.SampleRate > 0 {
		attrs = append(
			attrs,
			tracing.AttrRequestSampleRate.Int(t.attrs.SampleRate),
		)
	}
	return attrs
}

func (t *tracingGeneration) GenerateAudio(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (*Response, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartAudioSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)
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
	start := time.Now()
	ctx, span := t.startSpan(ctx)
	span.SetAttributes(tracing.AttrInputCount.Int(len(text)))
	innerCh, err := t.inner.StreamAudio(ctx, text, options...)
	return t.traceStream(ctx, span, start, "generate_audio", innerCh, err)
}

// startSpan opens an audio span for the wrapped model, carrying the
// construction-time attributes.
func (t *tracingGeneration) startSpan(
	ctx context.Context,
) (context.Context, tracing.Span) {
	m := t.inner.Model()
	return tracing.StartAudioSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)
}

// traceStream finishes span and records metrics for operation, timed from
// start, once innerCh is drained, forwarding every chunk to the returned
// channel. When err is set it records the failure at once and returns err.
func (t *tracingGeneration) traceStream(
	ctx context.Context,
	span tracing.Span,
	start time.Time,
	operation string,
	innerCh <-chan Chunk,
	err error,
) (<-chan Chunk, error) {
	m := t.inner.Model()
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx, operation, m.APIModel, string(m.Provider),
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
			ctx, operation, m.APIModel, string(m.Provider),
			time.Since(start), 0, 0, nil,
		)
	}()
	return outCh, nil
}

// tracedForcedAlignment adds a traced GenerateForcedAlignment to a tracing
// wrapper whose inner client implements ForcedAlignmentProvider.
type tracedForcedAlignment struct {
	t   *tracingGeneration
	fap ForcedAlignmentProvider
}

// GenerateForcedAlignment traces and forwards to the inner client's
// GenerateForcedAlignment.
func (f *tracedForcedAlignment) GenerateForcedAlignment(
	ctx context.Context,
	audioFile []byte,
	transcript string,
) (*ForcedAlignmentData, error) {
	m := f.t.inner.Model()
	start := time.Now()
	ctx, span := f.t.startSpan(ctx)
	defer span.End()
	span.SetAttributes(tracing.AttrInputCount.Int(len(transcript)))

	resp, err := f.fap.GenerateForcedAlignment(ctx, audioFile, transcript)
	if err != nil {
		tracing.SetError(span, err)
	}
	tracing.RecordMetrics(
		ctx, "forced_alignment", m.APIModel, string(m.Provider),
		time.Since(start), 0, 0, err,
	)
	return resp, err
}

// tracedStreamingText adds a traced StreamAudioFromText to a tracing wrapper
// whose inner client implements StreamingTextProvider.
type tracedStreamingText struct {
	t   *tracingGeneration
	stp StreamingTextProvider
}

// StreamAudioFromText traces and forwards to the inner client's
// StreamAudioFromText.
func (s *tracedStreamingText) StreamAudioFromText(
	ctx context.Context,
	textIn <-chan string,
	options ...GenerationOption,
) (<-chan Chunk, error) {
	start := time.Now()
	ctx, span := s.t.startSpan(ctx)
	innerCh, err := s.stp.StreamAudioFromText(ctx, textIn, options...)
	return s.t.traceStream(ctx, span, start, "stream_audio_from_text", innerCh, err)
}

// tracedDialogue adds traced dialogue methods to a tracing wrapper whose inner
// client implements DialogueProvider.
type tracedDialogue struct {
	t  *tracingGeneration
	dp DialogueProvider
}

// GenerateDialogue traces and forwards to the inner client's GenerateDialogue.
func (d *tracedDialogue) GenerateDialogue(
	ctx context.Context,
	turns []DialogueTurn,
	options ...GenerationOption,
) (*Response, error) {
	m := d.t.inner.Model()
	start := time.Now()
	ctx, span := d.t.startSpan(ctx)
	defer span.End()
	chars := 0
	for _, turn := range turns {
		chars += len(turn.Text)
	}
	span.SetAttributes(tracing.AttrInputCount.Int(chars))

	resp, err := d.dp.GenerateDialogue(ctx, turns, options...)
	if err != nil {
		tracing.SetError(span, err)
	} else {
		tracing.SetResponseAttrs(span,
			tracing.AttrUsageCharacters.Int64(resp.Usage.Characters),
		)
	}
	tracing.RecordMetrics(
		ctx, "generate_dialogue", m.APIModel, string(m.Provider),
		time.Since(start), 0, 0, err,
	)
	return resp, err
}

// StreamDialogue traces and forwards to the inner client's StreamDialogue.
func (d *tracedDialogue) StreamDialogue(
	ctx context.Context,
	voiceIDs []string,
	turns <-chan DialogueTurn,
	options ...GenerationOption,
) (<-chan Chunk, error) {
	start := time.Now()
	ctx, span := d.t.startSpan(ctx)
	innerCh, err := d.dp.StreamDialogue(ctx, voiceIDs, turns, options...)
	return d.t.traceStream(ctx, span, start, "stream_dialogue", innerCh, err)
}
