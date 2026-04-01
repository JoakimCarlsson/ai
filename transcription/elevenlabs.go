package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type elevenLabsOptions struct {
	diarize              *bool
	numSpeakers          *int
	timestampGranularity string
	tagAudioEvents       *bool
}

// ElevenLabsOption configures ElevenLabs Scribe-specific transcription behavior.
type ElevenLabsOption func(*elevenLabsOptions)

type elevenLabsClient struct {
	providerOptions transcriptionClientOptions
	options         elevenLabsOptions
	httpClient      *http.Client
	baseURL         string
}

// ElevenLabsClient is the ElevenLabs Scribe implementation of SpeechToTextClient.
type ElevenLabsClient SpeechToTextClient

type elScribeResponse struct {
	LanguageCode        string  `json:"language_code"`
	LanguageProbability float64 `json:"language_probability"`
	Text                string  `json:"text"`
	Words               []struct {
		Text      string  `json:"text"`
		Start     float64 `json:"start"`
		End       float64 `json:"end"`
		Type      string  `json:"type"`
		SpeakerID string  `json:"speaker_id,omitempty"`
	} `json:"words"`
}

func newElevenLabsClient(
	opts transcriptionClientOptions,
) ElevenLabsClient {
	elOpts := elevenLabsOptions{}
	for _, o := range opts.elevenLabsOptions {
		o(&elOpts)
	}

	timeout := 120 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &elevenLabsClient{
		providerOptions: opts,
		options:         elOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.elevenlabs.io/v1",
	}
}

func (e *elevenLabsClient) transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField(
		"model_id",
		e.providerOptions.model.APIModel,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to write model_id field: %w",
			err,
		)
	}

	if opts.Language != "" {
		if err := writer.WriteField(
			"language_code",
			opts.Language,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to write language_code field: %w",
				err,
			)
		}
	}

	if e.options.diarize != nil && *e.options.diarize {
		if err := writer.WriteField(
			"diarize",
			"true",
		); err != nil {
			return nil, fmt.Errorf(
				"failed to write diarize field: %w",
				err,
			)
		}
	}

	if e.options.numSpeakers != nil {
		if err := writer.WriteField(
			"num_speakers",
			fmt.Sprintf("%d", *e.options.numSpeakers),
		); err != nil {
			return nil, fmt.Errorf(
				"failed to write num_speakers field: %w",
				err,
			)
		}
	}

	granularity := "word"
	if e.options.timestampGranularity != "" {
		granularity = e.options.timestampGranularity
	}
	if err := writer.WriteField(
		"timestamps_granularity",
		granularity,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to write timestamps field: %w",
			err,
		)
	}

	filename := "audio.mp3"
	if opts.Filename != "" {
		filename = opts.Filename
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create form file: %w",
			err,
		)
	}
	if _, err := part.Write(audioFile); err != nil {
		return nil, fmt.Errorf(
			"failed to write audio data: %w",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf(
			"failed to close multipart writer: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		e.baseURL+"/speech-to-text",
		&buf,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create STT request: %w",
			err,
		)
	}

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)
	req.Header.Set(
		"xi-api-key",
		e.providerOptions.apiKey,
	)

	resp, err := e.httpClient.Do(req)
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

	var elResp elScribeResponse
	if err := json.Unmarshal(body, &elResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal STT response: %w",
			err,
		)
	}

	return e.mapResponse(&elResp), nil
}

func (e *elevenLabsClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return nil, fmt.Errorf(
		"elevenlabs scribe does not support translation",
	)
}

func (e *elevenLabsClient) mapResponse(
	elResp *elScribeResponse,
) *Response {
	result := &Response{
		Text:     elResp.Text,
		Language: elResp.LanguageCode,
		Model:    e.providerOptions.model.APIModel,
	}

	var words []Word
	for _, w := range elResp.Words {
		if w.Type != "word" {
			continue
		}
		words = append(words, Word{
			Word:  w.Text,
			Start: w.Start,
			End:   w.End,
		})
	}
	result.Words = words

	return result
}

// WithElevenLabsDiarize enables speaker diarization.
func WithElevenLabsDiarize(
	enabled bool,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.diarize = &enabled
	}
}

// WithElevenLabsNumSpeakers sets the expected number of speakers (0-32).
func WithElevenLabsNumSpeakers(
	n int,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.numSpeakers = &n
	}
}

// WithElevenLabsTimestampGranularity sets timestamp level ("none", "word", "character").
func WithElevenLabsTimestampGranularity(
	granularity string,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.timestampGranularity = granularity
	}
}

// WithElevenLabsTagAudioEvents enables audio event detection (laughter, music, etc.).
func WithElevenLabsTagAudioEvents(
	enabled bool,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.tagAudioEvents = &enabled
	}
}
