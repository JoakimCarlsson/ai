// Package berget provides a Berget AI implementation of the [stt.SpeechToText]
// interface.
//
// Berget exposes an OpenAI-compatible /v1/audio/transcriptions endpoint, but
// its verbose_json response nests segments under a "segments" object
// (segments.segments and segments.word_segments) rather than the flat arrays
// OpenAI returns, so this is a dedicated client rather than a thin wrapper over
// stt/openai. See [github.com/joakimcarlsson/ai/model] for the catalog
// (BergetTranscriptionModels) and pricing (EUR).
package berget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
)

// DefaultBaseURL is the canonical Berget AI OpenAI-compatible API endpoint.
const DefaultBaseURL = "https://api.berget.ai/v1"

// Options configures the Berget speech-to-text client.
type Options struct {
	apiKey   string
	model    model.TranscriptionModel
	timeout  *time.Duration
	baseURL  string
	language string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Berget.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the transcription model.
func WithModel(m model.TranscriptionModel) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithBaseURL points the client at a custom endpoint (defaults to
// https://api.berget.ai/v1).
func WithBaseURL(baseURL string) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithLanguage sets the default language (ISO-639-1) for transcription.
// Per-call [stt.WithLanguage] overrides this when supplied.
func WithLanguage(language string) Option {
	return func(o *Options) { o.language = language }
}

// Client implements [stt.SpeechToText] against the Berget transcription API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewSpeechToText constructs a Berget AI speech-to-text client. The returned
// [stt.SpeechToText] is wrapped with [stt.WithTracing], so callers always get
// tracing spans and metrics.
func NewSpeechToText(opts ...Option) stt.SpeechToText {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	timeout := 60 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	baseURL := DefaultBaseURL
	if options.baseURL != "" {
		baseURL = options.baseURL
	}

	return stt.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
	}, stt.TracingAttrs{
		Language: options.language,
	})
}

// Model returns the configured transcription model.
func (c *Client) Model() model.TranscriptionModel { return c.options.model }

// SupportsStreaming reports false; Berget's transcription endpoint does not
// support real-time streaming.
func (c *Client) SupportsStreaming() bool { return false }

// StreamTranscribe returns [stt.ErrStreamingNotSupported].
func (c *Client) StreamTranscribe(
	_ context.Context,
	_ <-chan []byte,
	_ ...stt.Option,
) (<-chan stt.StreamResult, error) {
	return nil, stt.ErrStreamingNotSupported
}

// Transcribe converts audio to text in the same language as the audio.
func (c *Client) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	return c.post(ctx, "/audio/transcriptions", audioFile, options...)
}

// Translate converts audio to English text regardless of the source language.
func (c *Client) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	return c.post(ctx, "/audio/translations", audioFile, options...)
}

func (c *Client) post(
	ctx context.Context,
	path string,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	opts := stt.Options{Filename: "audio.mp3", ResponseFormat: "verbose_json"}
	for _, opt := range options {
		opt(&opts)
	}

	lang := c.options.language
	if opts.Language != "" {
		lang = opts.Language
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	fields := map[string]string{
		"model":           c.options.model.APIModel,
		"response_format": opts.ResponseFormat,
		"language":        lang,
		"prompt":          opts.Prompt,
	}
	if opts.Temperature != nil {
		fields["temperature"] = strconv.FormatFloat(*opts.Temperature, 'f', -1, 64)
	}
	for k, v := range fields {
		if v == "" {
			continue
		}
		if err := w.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("failed to write %s field: %w", k, err)
		}
	}
	for _, g := range opts.TimestampGranularities {
		if err := w.WriteField("timestamp_granularities[]", g); err != nil {
			return nil, fmt.Errorf("failed to write timestamp granularity: %w", err)
		}
	}

	part, err := w.CreateFormFile("file", opts.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create file part: %w", err)
	}
	if _, err := part.Write(audioFile); err != nil {
		return nil, fmt.Errorf("failed to write audio: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcription request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make transcription request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcription response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"transcription API request failed with status %d: %s",
			resp.StatusCode, string(respBody),
		)
	}

	return c.parse(respBody), nil
}

// parse decodes Berget's transcription response. Top-level text/language/
// duration always populate; segments are decoded leniently so a body without
// (or with an unexpected) "segments" shape still yields the transcript.
func (c *Client) parse(body []byte) *stt.Response {
	var raw struct {
		Text     string          `json:"text"`
		Language string          `json:"language"`
		Duration float64         `json:"duration"`
		Segments json.RawMessage `json:"segments"`
	}
	_ = json.Unmarshal(body, &raw)

	result := &stt.Response{
		Text:     raw.Text,
		Language: raw.Language,
		Duration: raw.Duration,
		Model:    c.options.model.APIModel,
	}

	var seg struct {
		Segments []struct {
			Start      float64 `json:"start"`
			End        float64 `json:"end"`
			Text       string  `json:"text"`
			AvgLogprob float64 `json:"avg_logprob"`
		} `json:"segments"`
		WordSegments []struct {
			Word  string  `json:"word"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"word_segments"`
	}
	if len(raw.Segments) > 0 && json.Unmarshal(raw.Segments, &seg) == nil {
		for i, s := range seg.Segments {
			result.Segments = append(result.Segments, stt.Segment{
				ID:         i,
				Start:      s.Start,
				End:        s.End,
				Text:       s.Text,
				AvgLogprob: s.AvgLogprob,
			})
		}
		for _, word := range seg.WordSegments {
			result.Words = append(result.Words, stt.Word{
				Word:  word.Word,
				Start: word.Start,
				End:   word.End,
			})
		}
	}

	return result
}
