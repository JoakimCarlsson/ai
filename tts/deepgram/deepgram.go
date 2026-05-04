// Package deepgram provides a Deepgram Aura implementation of the [tts.Generation] interface.
package deepgram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
)

const (
	defaultBaseURL  = "https://api.deepgram.com/v1"
	defaultModelStr = "aura-asteria-en"
)

// Options configures the Deepgram TTS client.
type Options struct {
	apiKey     string
	model      model.AudioModel
	timeout    *time.Duration
	baseURL    string
	modelName  string
	encoding   string
	container  string
	sampleRate int
	bitRate    int
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Deepgram.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the TTS model from the model package.
func WithModel(m model.AudioModel) Option { return func(o *Options) { o.model = m } }

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithBaseURL sets a custom base URL for the Deepgram API.
func WithBaseURL(baseURL string) Option { return func(o *Options) { o.baseURL = baseURL } }

// WithModelName overrides the model identifier (e.g. "aura-2-thalia-en"). Takes
// precedence over [WithModel].
func WithModelName(name string) Option { return func(o *Options) { o.modelName = name } }

// WithEncoding sets the audio encoding (e.g. "mp3", "linear16", "mulaw", "alaw",
// "opus", "flac", "aac"). Default is "mp3".
func WithEncoding(encoding string) Option {
	return func(o *Options) { o.encoding = encoding }
}

// WithContainer sets the audio container (e.g. "wav", "ogg", "none").
func WithContainer(container string) Option {
	return func(o *Options) { o.container = container }
}

// WithSampleRate sets the audio sample rate in Hz.
func WithSampleRate(rate int) Option { return func(o *Options) { o.sampleRate = rate } }

// WithBitRate sets the audio bit rate in bps for compressed encodings.
func WithBitRate(rate int) Option { return func(o *Options) { o.bitRate = rate } }

// Client implements [tts.Generation] against the Deepgram Aura TTS API.
type Client struct {
	options    Options
	httpClient *http.Client
	resolved   string
}

// NewGeneration constructs a Deepgram Aura TTS client.
func NewGeneration(opts ...Option) tts.Generation {
	options := Options{baseURL: defaultBaseURL}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	resolved := defaultModelStr
	if options.model.APIModel != "" {
		resolved = options.model.APIModel
	}
	if options.modelName != "" {
		resolved = options.modelName
	}

	return tts.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		resolved:   resolved,
	}, tts.TracingAttrs{
		OutputFormat: options.encoding,
		SampleRate:   options.sampleRate,
	})
}

// Model returns the configured TTS model.
func (c *Client) Model() model.AudioModel { return c.options.model }

type ttsRequest struct {
	Text string `json:"text"`
}

type errorResponse struct {
	ErrCode string `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

func (c *Client) buildURL() string {
	q := url.Values{}
	q.Set("model", c.resolved)
	if c.options.encoding != "" {
		q.Set("encoding", c.options.encoding)
	}
	if c.options.container != "" {
		q.Set("container", c.options.container)
	}
	if c.options.sampleRate != 0 {
		q.Set("sample_rate", strconv.Itoa(c.options.sampleRate))
	}
	if c.options.bitRate != 0 {
		q.Set("bit_rate", strconv.Itoa(c.options.bitRate))
	}
	return fmt.Sprintf("%s/speak?%s", c.options.baseURL, q.Encode())
}

func (c *Client) newRequest(ctx context.Context, text string) (*http.Request, error) {
	body, err := json.Marshal(ttsRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.buildURL(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.options.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// GenerateAudio creates audio from text and returns the complete audio data.
func (c *Client) GenerateAudio(
	ctx context.Context,
	text string,
	_ ...tts.GenerationOption,
) (*tts.Response, error) {
	req, err := c.newRequest(ctx, text)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	charCount := int64(len(text))
	if v := resp.Header.Get("dg-char-count"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			charCount = n
		}
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mpeg"
	}

	return &tts.Response{
		AudioData:   audioData,
		ContentType: contentType,
		Usage:       tts.Usage{Characters: charCount},
		Model:       c.resolved,
	}, nil
}

// StreamAudio reads the Deepgram response body in 4KB chunks for real-time playback.
func (c *Client) StreamAudio(
	ctx context.Context,
	text string,
	_ ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	req, err := c.newRequest(ctx, text)
	if err != nil {
		return nil, err
	}

	chunkChan := make(chan tts.Chunk, 10)

	go func() {
		defer close(chunkChan)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			chunkChan <- tts.Chunk{Error: fmt.Errorf("request failed: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			chunkChan <- tts.Chunk{Error: c.parseError(resp)}
			return
		}

		buffer := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])
				chunkChan <- tts.Chunk{Data: data, Done: false}
			}

			if err == io.EOF {
				chunkChan <- tts.Chunk{Done: true}
				return
			}

			if err != nil {
				chunkChan <- tts.Chunk{Error: fmt.Errorf("stream read error: %w", err)}
				return
			}

			select {
			case <-ctx.Done():
				chunkChan <- tts.Chunk{Error: ctx.Err()}
				return
			default:
			}
		}
	}()

	return chunkChan, nil
}

// ListVoices returns the static set of Aura voices known to deps/ai. Deepgram
// does not expose a public list-voices endpoint.
func (c *Client) ListVoices(_ context.Context) ([]tts.Voice, error) {
	return []tts.Voice{
		{VoiceID: "aura-2-thalia-en", Name: "Thalia", Category: "aura-2"},
		{VoiceID: "aura-2-andromeda-en", Name: "Andromeda", Category: "aura-2"},
		{VoiceID: "aura-2-helena-en", Name: "Helena", Category: "aura-2"},
		{VoiceID: "aura-asteria-en", Name: "Asteria", Category: "aura"},
		{VoiceID: "aura-luna-en", Name: "Luna", Category: "aura"},
		{VoiceID: "aura-stella-en", Name: "Stella", Category: "aura"},
		{VoiceID: "aura-zeus-en", Name: "Zeus", Category: "aura"},
	}, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("audio generation failed with status %d", resp.StatusCode)
	}

	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.ErrMsg != "" {
		return fmt.Errorf("audio generation failed: %s", errResp.ErrMsg)
	}

	return fmt.Errorf("audio generation failed with status %d: %s", resp.StatusCode, string(body))
}
