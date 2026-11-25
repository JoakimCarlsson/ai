// Package openai provides an OpenAI implementation of the [tts.Generation] interface.
//
// OpenAI has deprecated every model this endpoint serves (tts-1, tts-1-hd and
// gpt-4o-mini-tts) and shuts them down on 2027-01-06; its replacement,
// gpt-realtime-2.1-mini, is served only by the Realtime API, which this
// package does not cover.
package openai

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/joakimcarlsson/ai/tts"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
)

// Options configures the OpenAI TTS client.
type Options struct {
	apiKey        string
	model         tts.AudioModel
	timeout       *time.Duration
	baseURL       string
	speed         *float64
	voice         string
	customVoiceID string
	instructions  string
	outputFormat  string

	extraBodyFields map[string]any
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with OpenAI.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the TTS model.
func WithModel(
	m tts.AudioModel,
) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithBaseURL points the client at a custom OpenAI-compatible endpoint.
func WithBaseURL(
	baseURL string,
) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithSpeed sets the speed of generated audio (0.25 to 4.0).
func WithSpeed(
	speed float64,
) Option {
	return func(o *Options) { o.speed = &speed }
}

// WithVoice sets a built-in voice: alloy, ash, ballad, cedar, coral, echo,
// fable, marin, nova, onyx, sage, shimmer or verse. tts-1 and tts-1-hd accept
// only alloy, ash, coral, echo, fable, nova, onyx, sage and shimmer. Without a
// voice the client sends alloy to OpenAI and no voice to a custom base URL,
// leaving the choice to that endpoint.
func WithVoice(
	name string,
) Option {
	return func(o *Options) { o.voice = name }
}

// WithCustomVoiceID selects a custom voice by its ID (e.g. "voice_1234").
// Takes precedence over [WithVoice].
func WithCustomVoiceID(id string) Option {
	return func(o *Options) { o.customVoiceID = id }
}

// WithInstructions steers how the speech is delivered: tone, accent, pacing,
// emotion. Not supported by tts-1 or tts-1-hd.
func WithInstructions(instructions string) Option {
	return func(o *Options) { o.instructions = instructions }
}

// WithOutputFormat sets the audio response format (e.g. "mp3", "opus", "aac",
// "flac", "wav", "pcm"). If unset, OpenAI's API default (MP3) applies.
func WithOutputFormat(
	format string,
) Option {
	return func(o *Options) { o.outputFormat = format }
}

// WithRequestJSONField injects an arbitrary top-level field into the request
// body via the SDK's WithJSONSet. It is the shared mechanism OpenAI-compatible
// providers use to express vendor features that are absent from the OpenAI wire
// schema, such as OpenRouter's provider routing object. Injected fields are only
// meaningful when a custom base URL targets a provider that accepts them.
func WithRequestJSONField(
	key string,
	value any,
) Option {
	return func(o *Options) {
		if o.extraBodyFields == nil {
			o.extraBodyFields = make(map[string]any)
		}
		o.extraBodyFields[key] = value
	}
}

// Client implements [tts.Generation] against the OpenAI Audio Speech API.
type Client struct {
	options Options
	client  openaisdk.Client
}

// NewGeneration constructs an OpenAI TTS client. The returned [tts.Generation]
// is wrapped with [tts.WithTracing], so callers always get tracing spans and metrics.
func NewGeneration(opts ...Option) tts.Generation {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{}
	if options.apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(options.apiKey))
	}
	if options.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(options.baseURL))
	}

	return tts.WithTracing(&Client{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, tts.TracingAttrs{
		Voice:        options.voice,
		OutputFormat: options.outputFormat,
		Speed:        options.speed,
	})
}

// Model returns the configured TTS model.
func (c *Client) Model() tts.AudioModel { return c.options.model }

func (c *Client) requestOptions() []option.RequestOption {
	var opts []option.RequestOption
	for k, v := range c.options.extraBodyFields {
		opts = append(opts, option.WithJSONSet(k, v))
	}
	return opts
}

// newParams builds the speech request for text from the client options and
// the per-call opts, returning it with the response format it asks for.
func (c *Client) newParams(
	text string,
	opts tts.GenerationOptions,
) openaisdk.AudioSpeechNewParams {
	params := openaisdk.AudioSpeechNewParams{
		Input: text,
		Model: openaisdk.SpeechModel(c.options.model.APIModel),
	}
	switch {
	case c.options.customVoiceID != "":
		params.Voice.OfAudioSpeechNewsVoiceID = &openaisdk.AudioSpeechNewParamsVoiceID{
			ID: c.options.customVoiceID,
		}
	case c.options.voice != "":
		params.Voice.OfString = param.NewOpt(c.options.voice)
	case c.options.baseURL == "":
		params.Voice.OfString = param.NewOpt("alloy")
	}

	outputFormat := c.options.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}
	if outputFormat != "" {
		params.ResponseFormat = openaisdk.AudioSpeechNewParamsResponseFormat(
			outputFormat,
		)
	}
	if c.options.speed != nil {
		params.Speed = param.NewOpt(*c.options.speed)
	}
	if c.options.instructions != "" {
		params.Instructions = param.NewOpt(c.options.instructions)
	}
	return params
}

// GenerateAudio creates audio from text and returns the complete audio data.
func (c *Client) GenerateAudio(
	ctx context.Context,
	text string,
	options ...tts.GenerationOption,
) (*tts.Response, error) {
	opts := tts.GenerationOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	params := c.newParams(text, opts)

	resp, err := c.client.Audio.Speech.New(ctx, params, c.requestOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio response: %w", err)
	}

	return &tts.Response{
		AudioData:   audioData,
		ContentType: resp.Header.Get("Content-Type"),
		Usage:       tts.Usage{Characters: int64(len(text))},
		Model:       c.options.model.APIModel,
	}, nil
}

// streamChunkSize is how many bytes of audio StreamAudio reads per chunk.
const streamChunkSize = 16 * 1024

// StreamAudio sends the speech request and emits the audio as it arrives,
// before generation has finished; pcm and wav give the lowest latency. A
// request failure is returned at once, a failure while reading arrives as a
// chunk error.
func (c *Client) StreamAudio(
	ctx context.Context,
	text string,
	options ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	opts := tts.GenerationOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	resp, err := c.client.Audio.Speech.New(
		ctx, c.newParams(text, opts), c.requestOptions()...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	ch := make(chan tts.Chunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		for {
			buf := make([]byte, streamChunkSize)
			n, readErr := io.ReadFull(resp.Body, buf)
			if n > 0 {
				select {
				case ch <- tts.Chunk{Data: buf[:n]}:
				case <-ctx.Done():
					return
				}
			}
			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				select {
				case ch <- tts.Chunk{Done: true}:
				case <-ctx.Done():
				}
				return
			}
			if readErr != nil {
				select {
				case ch <- tts.Chunk{Error: fmt.Errorf("failed to read audio stream: %w", readErr)}:
				case <-ctx.Done():
				}
				return
			}
		}
	}()
	return ch, nil
}

// ListVoices returns the OpenAI voice catalogue (static list — OpenAI does not
// expose a list-voices endpoint).
func (c *Client) ListVoices(_ context.Context) ([]tts.Voice, error) {
	return []tts.Voice{
		{VoiceID: "alloy", Name: "Alloy"},
		{VoiceID: "ash", Name: "Ash"},
		{VoiceID: "ballad", Name: "Ballad"},
		{VoiceID: "cedar", Name: "Cedar"},
		{VoiceID: "coral", Name: "Coral"},
		{VoiceID: "echo", Name: "Echo"},
		{VoiceID: "fable", Name: "Fable"},
		{VoiceID: "marin", Name: "Marin"},
		{VoiceID: "onyx", Name: "Onyx"},
		{VoiceID: "nova", Name: "Nova"},
		{VoiceID: "sage", Name: "Sage"},
		{VoiceID: "shimmer", Name: "Shimmer"},
		{VoiceID: "verse", Name: "Verse"},
	}, nil
}
