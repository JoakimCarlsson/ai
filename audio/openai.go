package audio

import (
	"context"
	"fmt"
	"io"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

type openaiAudioOptions struct {
	baseURL      string
	speed        *float64
	voice        string
	outputFormat string
}

// OpenAIAudioOption configures OpenAI-specific TTS behavior.
type OpenAIAudioOption func(*openaiAudioOptions)

type openaiClient struct {
	providerOptions audioGenerationClientOptions
	options         openaiAudioOptions
	client          openai.Client
}

// OpenAIClient is the OpenAI implementation of GenerationClient.
type OpenAIClient GenerationClient

func newOpenAIClient(
	opts audioGenerationClientOptions,
) OpenAIClient {
	openaiOpts := openaiAudioOptions{}
	for _, o := range opts.openaiAudioOptions {
		o(&openaiOpts)
	}

	clientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		clientOptions = append(
			clientOptions,
			option.WithAPIKey(opts.apiKey),
		)
	}
	if openaiOpts.baseURL != "" {
		clientOptions = append(
			clientOptions,
			option.WithBaseURL(openaiOpts.baseURL),
		)
	}

	return &openaiClient{
		providerOptions: opts,
		options:         openaiOpts,
		client:          openai.NewClient(clientOptions...),
	}
}

func (o *openaiClient) generate(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (*Response, error) {
	opts := GenerationOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	voice := o.options.voice
	if voice == "" {
		voice = "alloy"
	}
	params := openai.AudioSpeechNewParams{
		Input: text,
		Model: openai.SpeechModel(
			o.providerOptions.model.APIModel,
		),
		Voice: openai.AudioSpeechNewParamsVoice(voice),
	}

	outputFormat := o.options.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}
	if outputFormat != "" {
		params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat(
			outputFormat,
		)
	}
	if o.options.speed != nil {
		params.Speed = param.NewOpt(*o.options.speed)
	}

	resp, err := o.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate audio: %w",
			err,
		)
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read audio response: %w",
			err,
		)
	}

	contentType := resp.Header.Get("Content-Type")

	return &Response{
		AudioData:   audioData,
		ContentType: contentType,
		Usage: Usage{
			Characters: int64(len(text)),
		},
		Model: o.providerOptions.model.APIModel,
	}, nil
}

func (o *openaiClient) stream(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (<-chan Chunk, error) {
	resp, err := o.generate(ctx, text, options...)
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

func (o *openaiClient) listVoices(
	_ context.Context,
) ([]Voice, error) {
	voices := []Voice{
		{VoiceID: "alloy", Name: "Alloy"},
		{VoiceID: "ash", Name: "Ash"},
		{VoiceID: "ballad", Name: "Ballad"},
		{VoiceID: "coral", Name: "Coral"},
		{VoiceID: "echo", Name: "Echo"},
		{VoiceID: "fable", Name: "Fable"},
		{VoiceID: "onyx", Name: "Onyx"},
		{VoiceID: "nova", Name: "Nova"},
		{VoiceID: "sage", Name: "Sage"},
		{VoiceID: "shimmer", Name: "Shimmer"},
		{VoiceID: "verse", Name: "Verse"},
	}
	return voices, nil
}

// WithOpenAIBaseURL sets a custom base URL for the OpenAI API endpoint.
func WithOpenAIBaseURL(baseURL string) OpenAIAudioOption {
	return func(options *openaiAudioOptions) {
		options.baseURL = baseURL
	}
}

// WithOpenAISpeed sets the speed of generated audio (0.25 to 4.0).
func WithOpenAISpeed(speed float64) OpenAIAudioOption {
	return func(options *openaiAudioOptions) {
		options.speed = &speed
	}
}

// WithOpenAIVoice sets the voice used by every GenerateAudio / StreamAudio
// call on this client. Voice is set at construction time, like model — there
// is no per-call override. Valid values: alloy, ash, ballad, coral, echo,
// fable, onyx, nova, sage, shimmer, verse.
func WithOpenAIVoice(name string) OpenAIAudioOption {
	return func(options *openaiAudioOptions) {
		options.voice = name
	}
}

// WithOpenAIOutputFormat sets the audio response format (e.g. "mp3", "opus",
// "aac", "flac", "wav", "pcm") used by every GenerateAudio / StreamAudio call
// on this client. Set at construction time. If unset, OpenAI's API default
// (MP3) applies. Note: "pcm" returns 24kHz signed 16-bit little-endian raw PCM.
func WithOpenAIOutputFormat(format string) OpenAIAudioOption {
	return func(options *openaiAudioOptions) {
		options.outputFormat = format
	}
}
