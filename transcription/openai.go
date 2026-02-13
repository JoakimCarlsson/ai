package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type namedReader struct {
	reader io.Reader
	name   string
}

func (n *namedReader) Read(p []byte) (int, error) {
	return n.reader.Read(p)
}

func (n *namedReader) Name() string {
	return n.name
}

func (n *namedReader) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("stat not supported")
}

type openaiClient struct {
	providerOptions transcriptionClientOptions
	options         openaiOptions
	client          openai.Client
}

type OpenAIClient SpeechToTextClient

func newOpenAIClient(opts transcriptionClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{}
	for _, o := range opts.openaiOptions {
		o(&openaiOpts)
	}

	openaiClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		openaiClientOptions = append(
			openaiClientOptions,
			option.WithAPIKey(opts.apiKey),
		)
	}

	if openaiOpts.baseURL != "" {
		openaiClientOptions = append(
			openaiClientOptions,
			option.WithBaseURL(openaiOpts.baseURL),
		)
	}

	client := openai.NewClient(openaiClientOptions...)
	return &openaiClient{
		providerOptions: opts,
		options:         openaiOpts,
		client:          client,
	}
}

func (o *openaiClient) transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...TranscriptionOption,
) (*TranscriptionResponse, error) {
	opts := TranscriptionOptions{
		Filename: "audio.mp3",
	}
	for _, opt := range options {
		opt(&opts)
	}

	params := openai.AudioTranscriptionNewParams{
		Model: openai.AudioModel(o.providerOptions.model.APIModel),
		File:  &namedReader{reader: bytes.NewReader(audioFile), name: opts.Filename},
	}

	if opts.Language != "" {
		params.Language = openai.String(opts.Language)
	}

	if opts.Prompt != "" {
		params.Prompt = openai.String(opts.Prompt)
	}

	if opts.ResponseFormat != "" {
		params.ResponseFormat = openai.AudioResponseFormat(opts.ResponseFormat)
	} else {
		params.ResponseFormat = openai.AudioResponseFormat("verbose_json")
	}

	if opts.Temperature != nil {
		params.Temperature = openai.Float(*opts.Temperature)
	}

	if len(opts.TimestampGranularities) > 0 {
		params.TimestampGranularities = opts.TimestampGranularities
	}

	response, err := o.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	return o.convertTranscriptionResponse(response), nil
}

func (o *openaiClient) translate(
	ctx context.Context,
	audioFile []byte,
	options ...TranscriptionOption,
) (*TranscriptionResponse, error) {
	opts := TranscriptionOptions{
		Filename: "audio.mp3",
	}
	for _, opt := range options {
		opt(&opts)
	}

	params := openai.AudioTranslationNewParams{
		Model: openai.AudioModel(o.providerOptions.model.APIModel),
		File:  &namedReader{reader: bytes.NewReader(audioFile), name: opts.Filename},
	}

	if opts.Prompt != "" {
		params.Prompt = openai.String(opts.Prompt)
	}

	if opts.ResponseFormat != "" {
		switch opts.ResponseFormat {
		case "json":
			params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormatJSON
		case "text":
			params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormatText
		case "srt":
			params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormatSRT
		case "verbose_json":
			params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormatVerboseJSON
		case "vtt":
			params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormatVTT
		default:
			params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormatJSON
		}
	} else {
		params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormatJSON
	}

	if opts.Temperature != nil {
		params.Temperature = openai.Float(*opts.Temperature)
	}

	response, err := o.client.Audio.Translations.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to translate audio: %w", err)
	}

	return o.convertTranslationResponse(response), nil
}

func (o *openaiClient) convertTranslationResponse(response *openai.Translation) *TranscriptionResponse {
	return &TranscriptionResponse{
		Text:  response.Text,
		Model: o.providerOptions.model.APIModel,
	}
}

type verboseTranscription struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
	Segments []struct {
		ID               int     `json:"id"`
		Start            float64 `json:"start"`
		End              float64 `json:"end"`
		Text             string  `json:"text"`
		Tokens           []int   `json:"tokens"`
		Temperature      float64 `json:"temperature"`
		AvgLogprob       float64 `json:"avg_logprob"`
		CompressionRatio float64 `json:"compression_ratio"`
		NoSpeechProb     float64 `json:"no_speech_prob"`
	} `json:"segments"`
	Words []struct {
		Word  string  `json:"word"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"words"`
}

func (o *openaiClient) convertTranscriptionResponse(response *openai.Transcription) *TranscriptionResponse {
	result := &TranscriptionResponse{
		Text:  response.Text,
		Model: o.providerOptions.model.APIModel,
	}

	if response.Usage.Type == "tokens" {
		result.Usage.TotalTokens = response.Usage.TotalTokens
		result.Usage.InputTokens = response.Usage.InputTokens
		result.Usage.OutputTokens = response.Usage.OutputTokens
		result.Usage.AudioTokens = response.Usage.InputTokenDetails.AudioTokens
		result.Usage.TextTokens = response.Usage.InputTokenDetails.TextTokens
	} else if response.Usage.Type == "duration" {
		result.Usage.DurationSec = response.Usage.Seconds
	}

	raw := response.RawJSON()
	if raw != "" {
		var verbose verboseTranscription
		if err := json.Unmarshal([]byte(raw), &verbose); err == nil {
			result.Language = verbose.Language
			result.Duration = verbose.Duration
			for _, s := range verbose.Segments {
				result.Segments = append(result.Segments, TranscriptionSegment{
					ID:               s.ID,
					Start:            s.Start,
					End:              s.End,
					Text:             s.Text,
					Tokens:           s.Tokens,
					Temperature:      s.Temperature,
					AvgLogprob:       s.AvgLogprob,
					CompressionRatio: s.CompressionRatio,
					NoSpeechProb:     s.NoSpeechProb,
				})
			}
			for _, w := range verbose.Words {
				result.Words = append(result.Words, TranscriptionWord{
					Word:  w.Word,
					Start: w.Start,
					End:   w.End,
				})
			}
		}
	}

	return result
}
