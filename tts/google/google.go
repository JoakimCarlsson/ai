// Package google provides a Google Cloud Text-to-Speech implementation of the
// [tts.Generation] interface.
package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
)

const defaultBaseURL = "https://texttospeech.googleapis.com/v1"

// Options configures the Google Cloud TTS client.
type Options struct {
	apiKey       string
	model        model.AudioModel
	timeout      *time.Duration
	languageCode string
	ssmlGender   string
	voiceName    string
	outputFormat string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Google Cloud TTS.
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

// WithLanguageCode sets the BCP-47 language code for voice selection.
func WithLanguageCode(
	code string,
) Option {
	return func(o *Options) { o.languageCode = code }
}

// WithSSMLGender sets the voice gender ("MALE", "FEMALE", "NEUTRAL").
func WithSSMLGender(
	gender string,
) Option {
	return func(o *Options) { o.ssmlGender = gender }
}

// WithVoiceName sets a specific voice name (e.g., "en-US-Wavenet-D").
func WithVoiceName(
	name string,
) Option {
	return func(o *Options) { o.voiceName = name }
}

// WithOutputFormat sets the audio encoding (e.g. "MP3", "LINEAR16", "OGG_OPUS",
// "MULAW", "ALAW"). Default is "MP3".
func WithOutputFormat(format string) Option {
	return func(o *Options) { o.outputFormat = format }
}

// Client implements [tts.Generation] against the Google Cloud TTS API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewGeneration constructs a Google Cloud TTS client.
func NewGeneration(opts ...Option) tts.Generation {
	options := Options{languageCode: "en-US"}
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
		baseURL:    defaultBaseURL,
	}, tts.TracingAttrs{
		Voice:        options.voiceName,
		OutputFormat: options.outputFormat,
		Language:     options.languageCode,
	})
}

// Model returns the configured TTS model.
func (c *Client) Model() model.AudioModel { return c.options.model }

type ttsRequest struct {
	Input       ttsInput       `json:"input"`
	Voice       ttsVoice       `json:"voice"`
	AudioConfig ttsAudioConfig `json:"audioConfig"`
}

type ttsInput struct {
	Text string `json:"text"`
}

type ttsVoice struct {
	LanguageCode string `json:"languageCode"`
	Name         string `json:"name,omitempty"`
	SSMLGender   string `json:"ssmlGender,omitempty"`
}

type ttsAudioConfig struct {
	AudioEncoding string `json:"audioEncoding"`
}

type synthesizeResponse struct {
	AudioContent string `json:"audioContent"`
}

type voicesResponse struct {
	Voices []struct {
		LanguageCodes          []string `json:"languageCodes"`
		Name                   string   `json:"name"`
		SSMLGender             string   `json:"ssmlGender"`
		NaturalSampleRateHertz int      `json:"naturalSampleRateHertz"`
	} `json:"voices"`
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

	encoding := "MP3"
	if c.options.outputFormat != "" {
		encoding = c.options.outputFormat
	}
	if opts.OutputFormat != "" {
		encoding = opts.OutputFormat
	}

	voice := ttsVoice{LanguageCode: c.options.languageCode}
	if c.options.voiceName != "" {
		voice.Name = c.options.voiceName
	}
	if c.options.ssmlGender != "" {
		voice.SSMLGender = c.options.ssmlGender
	}

	reqBody := ttsRequest{
		Input:       ttsInput{Text: text},
		Voice:       voice,
		AudioConfig: ttsAudioConfig{AudioEncoding: encoding},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TTS request: %w", err)
	}

	reqURL := fmt.Sprintf(
		"%s/text:synthesize?key=%s",
		c.baseURL,
		c.options.apiKey,
	)
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		reqURL,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf(
			"TTS API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var gcResp synthesizeResponse
	if err := json.Unmarshal(body, &gcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TTS response: %w", err)
	}

	audioData, err := base64.StdEncoding.DecodeString(gcResp.AudioContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio content: %w", err)
	}

	return &tts.Response{
		AudioData:   audioData,
		ContentType: contentTypeForEncoding(encoding),
		Usage:       tts.Usage{Characters: int64(len(text))},
		Model:       c.options.model.APIModel,
	}, nil
}

// StreamAudio buffers Google's non-streaming response into a single chunk for API parity.
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

// ListVoices retrieves the list of available voices from Google Cloud TTS.
func (c *Client) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	reqURL := fmt.Sprintf("%s/voices?key=%s", c.baseURL, c.options.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create voices request: %w", err)
	}

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
		return nil, fmt.Errorf(
			"voices API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var gcResp voicesResponse
	if err := json.Unmarshal(body, &gcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal voices response: %w", err)
	}

	voices := make([]tts.Voice, len(gcResp.Voices))
	for i, v := range gcResp.Voices {
		voices[i] = tts.Voice{
			VoiceID:  v.Name,
			Name:     v.Name,
			Category: v.SSMLGender,
		}
	}
	return voices, nil
}

func contentTypeForEncoding(encoding string) string {
	switch encoding {
	case "MP3":
		return "audio/mpeg"
	case "LINEAR16":
		return "audio/wav"
	case "OGG_OPUS":
		return "audio/ogg"
	case "MULAW", "ALAW":
		return "audio/basic"
	default:
		return "audio/mpeg"
	}
}
