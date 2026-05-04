// Package openai provides an OpenAI Whisper implementation of the [stt.SpeechToText] interface.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Options configures the OpenAI speech-to-text client.
type Options struct {
	apiKey   string
	model    model.TranscriptionModel
	timeout  *time.Duration
	baseURL  string
	language string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with OpenAI.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) {
		o.apiKey = apiKey
	}
}

// WithModel selects the transcription model.
func WithModel(m model.TranscriptionModel) Option {
	return func(o *Options) {
		o.model = m
	}
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.timeout = &timeout
	}
}

// WithBaseURL points the client at a custom OpenAI-compatible endpoint.
func WithBaseURL(baseURL string) Option {
	return func(o *Options) {
		o.baseURL = baseURL
	}
}

// WithLanguage sets the default language (ISO-639-1) for transcription.
// Per-call [stt.WithLanguage] overrides this when supplied.
func WithLanguage(language string) Option {
	return func(o *Options) {
		o.language = language
	}
}

// Client implements [stt.SpeechToText] against the OpenAI Whisper API.
type Client struct {
	options Options
	client  openaisdk.Client
}

// NewSpeechToText constructs an OpenAI speech-to-text client. The returned
// [stt.SpeechToText] is wrapped with [stt.WithTracing], so callers always get
// tracing spans and metrics.
func NewSpeechToText(opts ...Option) stt.SpeechToText {
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

	return stt.WithTracing(&Client{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, stt.TracingAttrs{
		Language: options.language,
	})
}

// Model returns the configured transcription model.
func (c *Client) Model() model.TranscriptionModel {
	return c.options.model
}

// SupportsStreaming reports false; OpenAI's transcription endpoint does not
// support real-time streaming.
func (c *Client) SupportsStreaming() bool {
	return false
}

// StreamTranscribe returns [stt.ErrStreamingNotSupported].
func (c *Client) StreamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...stt.Option,
) (<-chan stt.StreamResult, error) {
	return nil, stt.ErrStreamingNotSupported
}

type namedReader struct {
	reader io.Reader
	name   string
}

func (n *namedReader) Read(p []byte) (int, error) { return n.reader.Read(p) }
func (n *namedReader) Name() string               { return n.name }
func (n *namedReader) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("stat not supported")
}

// Transcribe converts audio to text in the same language as the audio.
func (c *Client) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	opts := stt.Options{Filename: "audio.mp3"}
	for _, opt := range options {
		opt(&opts)
	}

	params := openaisdk.AudioTranscriptionNewParams{
		Model: openaisdk.AudioModel(c.options.model.APIModel),
		File: &namedReader{
			reader: bytes.NewReader(audioFile),
			name:   opts.Filename,
		},
	}

	lang := c.options.language
	if opts.Language != "" {
		lang = opts.Language
	}
	if lang != "" {
		params.Language = openaisdk.String(lang)
	}

	if opts.Prompt != "" {
		params.Prompt = openaisdk.String(opts.Prompt)
	}

	if opts.ResponseFormat != "" {
		params.ResponseFormat = openaisdk.AudioResponseFormat(opts.ResponseFormat)
	} else {
		params.ResponseFormat = openaisdk.AudioResponseFormat("verbose_json")
	}

	if opts.Temperature != nil {
		params.Temperature = openaisdk.Float(*opts.Temperature)
	}

	if len(opts.TimestampGranularities) > 0 {
		params.TimestampGranularities = opts.TimestampGranularities
	}

	response, err := c.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	return c.convertResponse(response), nil
}

// Translate converts audio to English text regardless of the source language.
func (c *Client) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	opts := stt.Options{Filename: "audio.mp3"}
	for _, opt := range options {
		opt(&opts)
	}

	params := openaisdk.AudioTranslationNewParams{
		Model: openaisdk.AudioModel(c.options.model.APIModel),
		File: &namedReader{
			reader: bytes.NewReader(audioFile),
			name:   opts.Filename,
		},
	}

	if opts.Prompt != "" {
		params.Prompt = openaisdk.String(opts.Prompt)
	}

	if opts.ResponseFormat != "" {
		switch opts.ResponseFormat {
		case "json":
			params.ResponseFormat = openaisdk.AudioTranslationNewParamsResponseFormatJSON
		case "text":
			params.ResponseFormat = openaisdk.AudioTranslationNewParamsResponseFormatText
		case "srt":
			params.ResponseFormat = openaisdk.AudioTranslationNewParamsResponseFormatSRT
		case "verbose_json":
			params.ResponseFormat = openaisdk.AudioTranslationNewParamsResponseFormatVerboseJSON
		case "vtt":
			params.ResponseFormat = openaisdk.AudioTranslationNewParamsResponseFormatVTT
		default:
			params.ResponseFormat = openaisdk.AudioTranslationNewParamsResponseFormatJSON
		}
	} else {
		params.ResponseFormat = openaisdk.AudioTranslationNewParamsResponseFormatJSON
	}

	if opts.Temperature != nil {
		params.Temperature = openaisdk.Float(*opts.Temperature)
	}

	response, err := c.client.Audio.Translations.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to translate audio: %w", err)
	}

	return &stt.Response{
		Text:  response.Text,
		Model: c.options.model.APIModel,
	}, nil
}

type verboseTranscription struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
	Segments []struct {
		ID               int     `json:"id"`
		Start            float64 `json:"start"`
		End              float64 `json:"end"`
		Text             string  `json:"text"`
		Tokens           []int   `json:"tokens"`
		Temperature      float64 `json:"temperature"`
		AvgLogprob       float64 `json:"avg_logprob"`
		CompressionRatio float64 `json:"compression_ratio"`
		NoSpeechProb     float64 `json:"no_speech_prob"`
	} `json:"segments"`
	Words []struct {
		Word  string  `json:"word"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"words"`
}

func (c *Client) convertResponse(response *openaisdk.Transcription) *stt.Response {
	result := &stt.Response{
		Text:  response.Text,
		Model: c.options.model.APIModel,
	}

	switch response.Usage.Type {
	case "tokens":
		result.Usage.TotalTokens = response.Usage.TotalTokens
		result.Usage.InputTokens = response.Usage.InputTokens
		result.Usage.OutputTokens = response.Usage.OutputTokens
		result.Usage.AudioTokens = response.Usage.InputTokenDetails.AudioTokens
		result.Usage.TextTokens = response.Usage.InputTokenDetails.TextTokens
	case "duration":
		result.Usage.DurationSec = response.Usage.Seconds
	}

	raw := response.RawJSON()
	if raw != "" {
		var verbose verboseTranscription
		if err := json.Unmarshal([]byte(raw), &verbose); err == nil {
			result.Language = verbose.Language
			result.Duration = verbose.Duration
			for _, s := range verbose.Segments {
				result.Segments = append(result.Segments, stt.Segment{
					ID:               s.ID,
					Start:            s.Start,
					End:              s.End,
					Text:             s.Text,
					Tokens:           s.Tokens,
					Temperature:      s.Temperature,
					AvgLogprob:       s.AvgLogprob,
					CompressionRatio: s.CompressionRatio,
					NoSpeechProb:     s.NoSpeechProb,
				})
			}
			for _, w := range verbose.Words {
				result.Words = append(result.Words, stt.Word{
					Word:  w.Word,
					Start: w.Start,
					End:   w.End,
				})
			}
		}
	}

	return result
}
