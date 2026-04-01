package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type azureSpeechOptions struct {
	region       string
	voiceName    string
	outputFormat string
}

// AzureSpeechOption configures Azure Speech Services TTS behavior.
type AzureSpeechOption func(*azureSpeechOptions)

type azureClient struct {
	providerOptions audioGenerationClientOptions
	options         azureSpeechOptions
	httpClient      *http.Client
}

// AzureClient is the Azure Speech implementation of GenerationClient.
type AzureClient GenerationClient

func newAzureClient(
	opts audioGenerationClientOptions,
) AzureClient {
	azOpts := azureSpeechOptions{
		region:       "eastus",
		voiceName:    "en-US-JennyNeural",
		outputFormat: "audio-24khz-160kbitrate-mono-mp3",
	}
	for _, o := range opts.azureSpeechOptions {
		o(&azOpts)
	}

	timeout := 30 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &azureClient{
		providerOptions: opts,
		options:         azOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (a *azureClient) generate(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (*Response, error) {
	opts := GenerationOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	voiceName := a.options.voiceName
	if opts.VoiceID != "" {
		voiceName = opts.VoiceID
	}

	outputFormat := a.options.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}

	ssml := fmt.Sprintf(
		`<speak version='1.0' xml:lang='en-US'>`+
			`<voice name='%s'>%s</voice></speak>`,
		voiceName,
		text,
	)

	ttsURL := fmt.Sprintf(
		"https://%s.tts.speech.microsoft.com/cognitiveservices/v1",
		a.options.region,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		ttsURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create TTS request: %w",
			err,
		)
	}

	req.Body = io.NopCloser(
		io.NewSectionReader(
			readerAtFromBytes([]byte(ssml)),
			0,
			int64(len(ssml)),
		),
	)
	req.ContentLength = int64(len(ssml))
	req.Header.Set(
		"Content-Type",
		"application/ssml+xml",
	)
	req.Header.Set(
		"Ocp-Apim-Subscription-Key",
		a.providerOptions.apiKey,
	)
	req.Header.Set(
		"X-Microsoft-OutputFormat",
		outputFormat,
	)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make TTS request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read TTS response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"TTS API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	return &Response{
		AudioData:   body,
		ContentType: resp.Header.Get("Content-Type"),
		Usage: Usage{
			Characters: int64(len(text)),
		},
		Model: a.providerOptions.model.APIModel,
	}, nil
}

func (a *azureClient) stream(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (<-chan Chunk, error) {
	resp, err := a.generate(ctx, text, options...)
	if err != nil {
		return nil, err
	}

	ch := make(chan Chunk, 1)
	go func() {
		defer close(ch)
		ch <- Chunk{Data: resp.AudioData}
		ch <- Chunk{Done: true}
	}()

	return ch, nil
}

func (a *azureClient) listVoices(
	ctx context.Context,
) ([]Voice, error) {
	voicesURL := fmt.Sprintf(
		"https://%s.tts.speech.microsoft.com/cognitiveservices/voices/list",
		a.options.region,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		voicesURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create voices request: %w",
			err,
		)
	}

	req.Header.Set(
		"Ocp-Apim-Subscription-Key",
		a.providerOptions.apiKey,
	)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list voices: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read voices response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"voices API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var azureVoices []struct {
		Name        string `json:"Name"`
		DisplayName string `json:"DisplayName"`
		ShortName   string `json:"ShortName"`
		Gender      string `json:"Gender"`
		Locale      string `json:"Locale"`
	}

	if err := json.Unmarshal(body, &azureVoices); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal voices response: %w",
			err,
		)
	}

	voices := make([]Voice, len(azureVoices))
	for i, v := range azureVoices {
		voices[i] = Voice{
			VoiceID:  v.ShortName,
			Name:     v.DisplayName,
			Category: v.Gender,
			Labels: map[string]string{
				"locale": v.Locale,
			},
		}
	}

	return voices, nil
}

type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(
	p []byte,
	off int64,
) (int, error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[off:])
	return n, nil
}

func readerAtFromBytes(data []byte) *bytesReaderAt {
	return &bytesReaderAt{data: data}
}

// WithAzureRegion sets the Azure region for the Speech Service endpoint.
func WithAzureRegion(
	region string,
) AzureSpeechOption {
	return func(options *azureSpeechOptions) {
		options.region = region
	}
}

// WithAzureVoiceName sets the default voice name (e.g., "en-US-JennyNeural").
func WithAzureVoiceName(
	name string,
) AzureSpeechOption {
	return func(options *azureSpeechOptions) {
		options.voiceName = name
	}
}

// WithAzureOutputFormat sets the output audio format (e.g., "audio-24khz-160kbitrate-mono-mp3").
func WithAzureOutputFormat(
	format string,
) AzureSpeechOption {
	return func(options *azureSpeechOptions) {
		options.outputFormat = format
	}
}
