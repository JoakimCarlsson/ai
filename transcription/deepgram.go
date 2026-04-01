package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type deepgramOptions struct {
	punctuate   *bool
	diarize     *bool
	smartFormat *bool
	language    string
}

// DeepgramOption configures Deepgram-specific transcription behavior.
type DeepgramOption func(*deepgramOptions)

type deepgramClient struct {
	providerOptions transcriptionClientOptions
	options         deepgramOptions
	httpClient      *http.Client
	baseURL         string
}

// DeepgramClient is the Deepgram implementation of SpeechToTextClient.
type DeepgramClient SpeechToTextClient

type deepgramResponse struct {
	Results struct {
		Channels []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
				Words      []struct {
					Word       string  `json:"word"`
					Start      float64 `json:"start"`
					End        float64 `json:"end"`
					Confidence float64 `json:"confidence"`
					Speaker    *int    `json:"speaker,omitempty"`
				} `json:"words"`
			} `json:"alternatives"`
		} `json:"channels"`
	} `json:"results"`
	Metadata struct {
		Duration  float64        `json:"duration"`
		Channels  int            `json:"channels"`
		ModelInfo map[string]any `json:"model_info"`
		RequestID string         `json:"request_id"`
	} `json:"metadata"`
}

func newDeepgramClient(
	opts transcriptionClientOptions,
) DeepgramClient {
	dgOpts := deepgramOptions{}
	for _, o := range opts.deepgramOptions {
		o(&dgOpts)
	}

	timeout := 120 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &deepgramClient{
		providerOptions: opts,
		options:         dgOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.deepgram.com/v1",
	}
}

func (d *deepgramClient) transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}

	params := url.Values{}
	params.Set(
		"model",
		d.providerOptions.model.APIModel,
	)

	lang := d.options.language
	if opts.Language != "" {
		lang = opts.Language
	}
	if lang != "" {
		params.Set("language", lang)
	}

	if d.options.punctuate != nil && *d.options.punctuate {
		params.Set("punctuate", "true")
	}
	if d.options.diarize != nil && *d.options.diarize {
		params.Set("diarize", "true")
	}
	if d.options.smartFormat != nil && *d.options.smartFormat {
		params.Set("smart_format", "true")
	}

	reqURL := d.baseURL + "/listen?" + params.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		reqURL,
		bytes.NewReader(audioFile),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create transcription request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "audio/mpeg")
	req.Header.Set(
		"Authorization",
		"Token "+d.providerOptions.apiKey,
	)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make transcription request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read transcription response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"transcription API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var dgResp deepgramResponse
	if err := json.Unmarshal(body, &dgResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal transcription response: %w",
			err,
		)
	}

	return d.mapResponse(&dgResp), nil
}

func (d *deepgramClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return nil, fmt.Errorf(
		"deepgram does not support translation",
	)
}

func (d *deepgramClient) mapResponse(
	dgResp *deepgramResponse,
) *Response {
	result := &Response{
		Duration: dgResp.Metadata.Duration,
		Model:    d.providerOptions.model.APIModel,
		Usage: Usage{
			DurationSec: dgResp.Metadata.Duration,
		},
	}

	if len(dgResp.Results.Channels) > 0 &&
		len(dgResp.Results.Channels[0].Alternatives) > 0 {
		alt := dgResp.Results.Channels[0].Alternatives[0]
		result.Text = alt.Transcript

		words := make([]Word, len(alt.Words))
		for i, w := range alt.Words {
			words[i] = Word{
				Word:  w.Word,
				Start: w.Start,
				End:   w.End,
			}
		}
		result.Words = words
	}

	return result
}

// WithDeepgramPunctuate enables automatic punctuation.
func WithDeepgramPunctuate(
	enabled bool,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.punctuate = &enabled
	}
}

// WithDeepgramDiarize enables speaker diarization.
func WithDeepgramDiarize(
	enabled bool,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.diarize = &enabled
	}
}

// WithDeepgramSmartFormat enables smart formatting.
func WithDeepgramSmartFormat(
	enabled bool,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.smartFormat = &enabled
	}
}

// WithDeepgramLanguage sets the default language for transcription.
func WithDeepgramLanguage(
	language string,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.language = language
	}
}
