// Package azure provides an Azure Speech Services implementation of the
// [tts.Generation] interface.
package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
)

// Options configures the Azure Speech client.
type Options struct {
	apiKey       string
	model        model.AudioModel
	timeout      *time.Duration
	region       string
	voiceName    string
	outputFormat string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Azure Speech Services.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the TTS model.
func WithModel(m model.AudioModel) Option { return func(o *Options) { o.model = m } }

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithRegion sets the Azure region for the Speech Service endpoint.
func WithRegion(region string) Option { return func(o *Options) { o.region = region } }

// WithVoiceName sets the default voice name (e.g., "en-US-JennyNeural").
func WithVoiceName(name string) Option { return func(o *Options) { o.voiceName = name } }

// WithOutputFormat sets the output audio format (e.g., "audio-24khz-160kbitrate-mono-mp3").
func WithOutputFormat(format string) Option {
	return func(o *Options) { o.outputFormat = format }
}

// Client implements [tts.Generation] against Azure Speech Services.
type Client struct {
	options    Options
	httpClient *http.Client
}

// NewGeneration constructs an Azure Speech TTS client.
func NewGeneration(opts ...Option) tts.Generation {
	options := Options{
		region:       "eastus",
		voiceName:    "en-US-JennyNeural",
		outputFormat: "audio-24khz-160kbitrate-mono-mp3",
	}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return tts.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
	}, tts.TracingAttrs{
		Voice:        options.voiceName,
		OutputFormat: options.outputFormat,
	})
}

// Model returns the configured TTS model.
func (c *Client) Model() model.AudioModel { return c.options.model }

// GenerateAudio synthesises speech from text via SSML.
func (c *Client) GenerateAudio(
	ctx context.Context,
	text string,
	options ...tts.GenerationOption,
) (*tts.Response, error) {
	opts := tts.GenerationOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	outputFormat := c.options.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}

	ssml := fmt.Sprintf(
		`<speak version='1.0' xml:lang='en-US'><voice name='%s'>%s</voice></speak>`,
		c.options.voiceName, text,
	)

	ttsURL := fmt.Sprintf(
		"https://%s.tts.speech.microsoft.com/cognitiveservices/v1",
		c.options.region,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", ttsURL, bytes.NewReader([]byte(ssml)))
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("Ocp-Apim-Subscription-Key", c.options.apiKey)
	req.Header.Set("X-Microsoft-OutputFormat", outputFormat)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make TTS request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read TTS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS API failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &tts.Response{
		AudioData:   body,
		ContentType: resp.Header.Get("Content-Type"),
		Usage:       tts.Usage{Characters: int64(len(text))},
		Model:       c.options.model.APIModel,
	}, nil
}

// StreamAudio buffers Azure's non-streaming response into a single chunk for API parity.
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

// ListVoices retrieves the list of available voices from Azure.
func (c *Client) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	voicesURL := fmt.Sprintf(
		"https://%s.tts.speech.microsoft.com/cognitiveservices/voices/list",
		c.options.region,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", voicesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create voices request: %w", err)
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list voices: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read voices response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("voices API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var azureVoices []struct {
		Name        string `json:"Name"`
		DisplayName string `json:"DisplayName"`
		ShortName   string `json:"ShortName"`
		Gender      string `json:"Gender"`
		Locale      string `json:"Locale"`
	}

	if err := json.Unmarshal(body, &azureVoices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal voices response: %w", err)
	}

	voices := make([]tts.Voice, len(azureVoices))
	for i, v := range azureVoices {
		voices[i] = tts.Voice{
			VoiceID:  v.ShortName,
			Name:     v.DisplayName,
			Category: v.Gender,
			Labels:   map[string]string{"locale": v.Locale},
		}
	}
	return voices, nil
}
