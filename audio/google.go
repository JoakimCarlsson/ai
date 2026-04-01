package audio

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

type googleCloudTTSOptions struct {
	languageCode string
	ssmlGender   string
	voiceName    string
}

// GoogleCloudTTSOption configures Google Cloud TTS behavior.
type GoogleCloudTTSOption func(*googleCloudTTSOptions)

type googleCloudClient struct {
	providerOptions audioGenerationClientOptions
	options         googleCloudTTSOptions
	httpClient      *http.Client
	baseURL         string
}

// GoogleCloudClient is the Google Cloud implementation of GenerationClient.
type GoogleCloudClient GenerationClient

type gcTTSRequest struct {
	Input       gcTTSInput       `json:"input"`
	Voice       gcTTSVoice       `json:"voice"`
	AudioConfig gcTTSAudioConfig `json:"audioConfig"`
}

type gcTTSInput struct {
	Text string `json:"text"`
}

type gcTTSVoice struct {
	LanguageCode string `json:"languageCode"`
	Name         string `json:"name,omitempty"`
	SSMLGender   string `json:"ssmlGender,omitempty"`
}

type gcTTSAudioConfig struct {
	AudioEncoding string `json:"audioEncoding"`
}

type gcTTSSynthesizeResponse struct {
	AudioContent string `json:"audioContent"`
}

type gcTTSVoicesResponse struct {
	Voices []struct {
		LanguageCodes          []string `json:"languageCodes"`
		Name                   string   `json:"name"`
		SSMLGender             string   `json:"ssmlGender"`
		NaturalSampleRateHertz int      `json:"naturalSampleRateHertz"`
	} `json:"voices"`
}

func newGoogleCloudClient(
	opts audioGenerationClientOptions,
) GoogleCloudClient {
	gcOpts := googleCloudTTSOptions{
		languageCode: "en-US",
	}
	for _, o := range opts.googleCloudTTSOptions {
		o(&gcOpts)
	}

	timeout := 30 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &googleCloudClient{
		providerOptions: opts,
		options:         gcOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://texttospeech.googleapis.com/v1",
	}
}

func (g *googleCloudClient) generate(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (*Response, error) {
	opts := GenerationOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	encoding := "MP3"
	if opts.OutputFormat != "" {
		encoding = opts.OutputFormat
	}

	voice := gcTTSVoice{
		LanguageCode: g.options.languageCode,
	}
	if g.options.voiceName != "" {
		voice.Name = g.options.voiceName
	}
	if opts.VoiceID != "" {
		voice.Name = opts.VoiceID
	}
	if g.options.ssmlGender != "" {
		voice.SSMLGender = g.options.ssmlGender
	}

	reqBody := gcTTSRequest{
		Input: gcTTSInput{Text: text},
		Voice: voice,
		AudioConfig: gcTTSAudioConfig{
			AudioEncoding: encoding,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal TTS request: %w",
			err,
		)
	}

	reqURL := fmt.Sprintf(
		"%s/text:synthesize?key=%s",
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
			"failed to create TTS request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
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

	var gcResp gcTTSSynthesizeResponse
	if err := json.Unmarshal(body, &gcResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal TTS response: %w",
			err,
		)
	}

	audioData, err := base64.StdEncoding.DecodeString(
		gcResp.AudioContent,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to decode audio content: %w",
			err,
		)
	}

	return &Response{
		AudioData:   audioData,
		ContentType: contentTypeForEncoding(encoding),
		Usage: Usage{
			Characters: int64(len(text)),
		},
		Model: g.providerOptions.model.APIModel,
	}, nil
}

func (g *googleCloudClient) stream(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (<-chan Chunk, error) {
	resp, err := g.generate(ctx, text, options...)
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

func (g *googleCloudClient) listVoices(
	ctx context.Context,
) ([]Voice, error) {
	reqURL := fmt.Sprintf(
		"%s/voices?key=%s",
		g.baseURL,
		g.providerOptions.apiKey,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		reqURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create voices request: %w",
			err,
		)
	}

	resp, err := g.httpClient.Do(req)
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

	var gcResp gcTTSVoicesResponse
	if err := json.Unmarshal(body, &gcResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal voices response: %w",
			err,
		)
	}

	voices := make([]Voice, len(gcResp.Voices))
	for i, v := range gcResp.Voices {
		voices[i] = Voice{
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

// WithGoogleCloudLanguageCode sets the BCP-47 language code for voice selection.
func WithGoogleCloudLanguageCode(
	code string,
) GoogleCloudTTSOption {
	return func(options *googleCloudTTSOptions) {
		options.languageCode = code
	}
}

// WithGoogleCloudSSMLGender sets the voice gender ("MALE", "FEMALE", "NEUTRAL").
func WithGoogleCloudSSMLGender(
	gender string,
) GoogleCloudTTSOption {
	return func(options *googleCloudTTSOptions) {
		options.ssmlGender = gender
	}
}

// WithGoogleCloudVoiceName sets a specific voice name (e.g., "en-US-Wavenet-D").
func WithGoogleCloudVoiceName(
	name string,
) GoogleCloudTTSOption {
	return func(options *googleCloudTTSOptions) {
		options.voiceName = name
	}
}
