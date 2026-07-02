// Package openai provides an OpenAI implementation of the [tts.Generation] interface.
package openai

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
)

// Options configures the OpenAI TTS client.
type Options struct {
	apiKey       string
	model        model.AudioModel
	timeout      *time.Duration
	baseURL      string
	speed        *float64
	voice        string
	outputFormat string
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
	m model.AudioModel,
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

// WithVoice sets the voice. Valid values: alloy, ash, ballad, coral, echo, fable,
// onyx, nova, sage, shimmer, verse.
func WithVoice(
	name string,
) Option {
	return func(o *Options) { o.voice = name }
}

// WithOutputFormat sets the audio response format (e.g. "mp3", "opus", "aac",
// "flac", "wav", "pcm"). If unset, OpenAI's API default (MP3) applies.
func WithOutputFormat(
	format string,
) Option {
	return func(o *Options) { o.outputFormat = format }
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
func (c *Client) Model() model.AudioModel { return c.options.model }

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

	voice := c.options.voice
	if voice == "" {
		voice = "alloy"
	}
	params := openaisdk.AudioSpeechNewParams{
		Input: text,
		Model: openaisdk.SpeechModel(c.options.model.APIModel),
		Voice: openaisdk.AudioSpeechNewParamsVoiceUnion{OfString: param.NewOpt(voice)},
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

	resp, err := c.client.Audio.Speech.New(ctx, params)
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

// StreamAudio buffers OpenAI's non-streaming response into a single chunk for
// API parity with vendors that support real streaming.
func (c *Client) StreamAudio(
	ctx context.Context,
	text string,
	options ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	resp, err := c.GenerateAudio(ctx, text, options...)
	if err != nil {
		return nil, err
	}

	ch := make(chan tts.Chunk, 1)
	go func() {
		defer close(ch)
		ch <- tts.Chunk{Data: resp.AudioData}
		ch <- tts.Chunk{Done: true}
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
		{VoiceID: "coral", Name: "Coral"},
		{VoiceID: "echo", Name: "Echo"},
		{VoiceID: "fable", Name: "Fable"},
		{VoiceID: "onyx", Name: "Onyx"},
		{VoiceID: "nova", Name: "Nova"},
		{VoiceID: "sage", Name: "Sage"},
		{VoiceID: "shimmer", Name: "Shimmer"},
		{VoiceID: "verse", Name: "Verse"},
	}, nil
}
