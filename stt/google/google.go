// Package google provides a Google Cloud Speech-to-Text implementation of the
// [stt.SpeechToText] interface.
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
	"github.com/joakimcarlsson/ai/stt"
)

const defaultBaseURL = "https://speech.googleapis.com/v1"

// Options configures the Google Cloud STT client.
type Options struct {
	apiKey       string
	model        model.TranscriptionModel
	timeout      *time.Duration
	encoding     string
	sampleRateHz int
	languageCode string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Google Cloud STT.
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

// WithEncoding sets the audio encoding format (e.g., "LINEAR16", "FLAC", "OGG_OPUS").
func WithEncoding(encoding string) Option {
	return func(o *Options) {
		o.encoding = encoding
	}
}

// WithSampleRate sets the sample rate in hertz.
func WithSampleRate(hz int) Option {
	return func(o *Options) {
		o.sampleRateHz = hz
	}
}

// WithLanguageCode sets the default BCP-47 language code (e.g., "en-US").
// Per-call [stt.WithLanguage] overrides this when supplied.
func WithLanguageCode(code string) Option {
	return func(o *Options) {
		o.languageCode = code
	}
}

// Client implements [stt.SpeechToText] against the Google Cloud Speech-to-Text API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewSpeechToText constructs a Google Cloud STT client. The returned
// [stt.SpeechToText] is wrapped with [stt.WithTracing], so callers always get
// tracing spans and metrics.
func NewSpeechToText(opts ...Option) stt.SpeechToText {
	options := Options{
		languageCode: "en-US",
	}
	for _, o := range opts {
		o(&options)
	}

	timeout := 120 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return stt.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
	}, stt.TracingAttrs{
		Language: options.languageCode,
	})
}

// Model returns the configured transcription model.
func (c *Client) Model() model.TranscriptionModel {
	return c.options.model
}

// SupportsStreaming reports false; this client does not implement streaming.
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

type request struct {
	Config requestConfig `json:"config"`
	Audio  requestAudio  `json:"audio"`
}

type requestConfig struct {
	Encoding                   string `json:"encoding,omitempty"`
	SampleRateHertz            int    `json:"sampleRateHertz,omitempty"`
	LanguageCode               string `json:"languageCode"`
	Model                      string `json:"model,omitempty"`
	EnableWordTimeOffsets      bool   `json:"enableWordTimeOffsets,omitempty"`
	EnableAutomaticPunctuation bool   `json:"enableAutomaticPunctuation,omitempty"`
}

type requestAudio struct {
	Content string `json:"content"`
}

type response struct {
	Results []struct {
		Alternatives []struct {
			Transcript string  `json:"transcript"`
			Confidence float64 `json:"confidence"`
			Words      []struct {
				StartTime string `json:"startTime"`
				EndTime   string `json:"endTime"`
				Word      string `json:"word"`
			} `json:"words"`
		} `json:"alternatives"`
	} `json:"results"`
	TotalBilledTime string `json:"totalBilledTime"`
}

// Transcribe converts audio to text in the given language.
func (c *Client) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	opts := stt.Options{}
	for _, opt := range options {
		opt(&opts)
	}

	langCode := c.options.languageCode
	if opts.Language != "" {
		langCode = opts.Language
	}

	cfg := requestConfig{
		LanguageCode:               langCode,
		EnableWordTimeOffsets:      true,
		EnableAutomaticPunctuation: true,
	}
	if c.options.encoding != "" {
		cfg.Encoding = c.options.encoding
	}
	if c.options.sampleRateHz > 0 {
		cfg.SampleRateHertz = c.options.sampleRateHz
	}

	apiModel := c.options.model.APIModel
	if apiModel != "" && apiModel != "default" {
		cfg.Model = apiModel
	}

	reqBody := request{
		Config: cfg,
		Audio: requestAudio{
			Content: base64.StdEncoding.EncodeToString(audioFile),
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal STT request: %w", err)
	}

	reqURL := fmt.Sprintf(
		"%s/speech:recognize?key=%s",
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
		return nil, fmt.Errorf("failed to create STT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make STT request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read STT response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"STT API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var gcResp response
	if err := json.Unmarshal(body, &gcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal STT response: %w", err)
	}

	return c.mapResponse(&gcResp, langCode), nil
}

// Translate is not supported by Google Cloud STT.
func (c *Client) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	return nil, fmt.Errorf("google cloud STT does not support translation")
}

func (c *Client) mapResponse(gcResp *response, language string) *stt.Response {
	result := &stt.Response{
		Language: language,
		Model:    c.options.model.APIModel,
	}

	var fullText string
	var allWords []stt.Word

	for _, r := range gcResp.Results {
		if len(r.Alternatives) == 0 {
			continue
		}
		alt := r.Alternatives[0]
		fullText += alt.Transcript

		for _, w := range alt.Words {
			allWords = append(allWords, stt.Word{
				Word:  w.Word,
				Start: parseDuration(w.StartTime),
				End:   parseDuration(w.EndTime),
			})
		}
	}

	result.Text = fullText
	result.Words = allWords

	return result
}

func parseDuration(s string) float64 {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d.Seconds()
}
