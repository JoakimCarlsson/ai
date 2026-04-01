package transcription

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type googleCloudSTTOptions struct {
	encoding     string
	sampleRateHz int
	languageCode string
}

// GoogleCloudSTTOption configures Google Cloud STT behavior.
type GoogleCloudSTTOption func(*googleCloudSTTOptions)

type googleCloudClient struct {
	providerOptions transcriptionClientOptions
	options         googleCloudSTTOptions
	httpClient      *http.Client
	baseURL         string
}

// GoogleCloudClient is the Google Cloud implementation of SpeechToTextClient.
type GoogleCloudClient SpeechToTextClient

type gcSTTRequest struct {
	Config gcSTTConfig `json:"config"`
	Audio  gcSTTAudio  `json:"audio"`
}

type gcSTTConfig struct {
	Encoding                   string `json:"encoding,omitempty"`
	SampleRateHertz            int    `json:"sampleRateHertz,omitempty"`
	LanguageCode               string `json:"languageCode"`
	Model                      string `json:"model,omitempty"`
	EnableWordTimeOffsets      bool   `json:"enableWordTimeOffsets,omitempty"`
	EnableAutomaticPunctuation bool   `json:"enableAutomaticPunctuation,omitempty"`
}

type gcSTTAudio struct {
	Content string `json:"content"`
}

type gcSTTResponse struct {
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

func newGoogleCloudClient(
	opts transcriptionClientOptions,
) GoogleCloudClient {
	gcOpts := googleCloudSTTOptions{
		languageCode: "en-US",
	}
	for _, o := range opts.googleCloudSTTOptions {
		o(&gcOpts)
	}

	timeout := 120 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &googleCloudClient{
		providerOptions: opts,
		options:         gcOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://speech.googleapis.com/v1",
	}
}

func (g *googleCloudClient) transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}

	langCode := g.options.languageCode
	if opts.Language != "" {
		langCode = opts.Language
	}

	config := gcSTTConfig{
		LanguageCode:               langCode,
		EnableWordTimeOffsets:      true,
		EnableAutomaticPunctuation: true,
	}
	if g.options.encoding != "" {
		config.Encoding = g.options.encoding
	}
	if g.options.sampleRateHz > 0 {
		config.SampleRateHertz = g.options.sampleRateHz
	}

	apiModel := g.providerOptions.model.APIModel
	if apiModel != "" && apiModel != "default" {
		config.Model = apiModel
	}

	reqBody := gcSTTRequest{
		Config: config,
		Audio: gcSTTAudio{
			Content: base64.StdEncoding.EncodeToString(
				audioFile,
			),
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal STT request: %w",
			err,
		)
	}

	reqURL := fmt.Sprintf(
		"%s/speech:recognize?key=%s",
		g.baseURL,
		g.providerOptions.apiKey,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		reqURL,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create STT request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make STT request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read STT response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"STT API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var gcResp gcSTTResponse
	if err := json.Unmarshal(body, &gcResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal STT response: %w",
			err,
		)
	}

	return g.mapResponse(&gcResp, langCode), nil
}

func (g *googleCloudClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return nil, fmt.Errorf(
		"google cloud STT does not support translation",
	)
}

func (g *googleCloudClient) mapResponse(
	gcResp *gcSTTResponse,
	language string,
) *Response {
	result := &Response{
		Language: language,
		Model:    g.providerOptions.model.APIModel,
	}

	var fullText string
	var allWords []Word

	for _, r := range gcResp.Results {
		if len(r.Alternatives) == 0 {
			continue
		}
		alt := r.Alternatives[0]
		fullText += alt.Transcript

		for _, w := range alt.Words {
			allWords = append(allWords, Word{
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

// WithGoogleCloudEncoding sets the audio encoding format (e.g., "LINEAR16", "FLAC", "OGG_OPUS").
func WithGoogleCloudEncoding(
	encoding string,
) GoogleCloudSTTOption {
	return func(options *googleCloudSTTOptions) {
		options.encoding = encoding
	}
}

// WithGoogleCloudSampleRate sets the sample rate in hertz.
func WithGoogleCloudSampleRate(
	hz int,
) GoogleCloudSTTOption {
	return func(options *googleCloudSTTOptions) {
		options.sampleRateHz = hz
	}
}

// WithGoogleCloudLanguageCode sets the default BCP-47 language code.
func WithGoogleCloudLanguageCode(
	code string,
) GoogleCloudSTTOption {
	return func(options *googleCloudSTTOptions) {
		options.languageCode = code
	}
}
