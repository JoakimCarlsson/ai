// Package openai provides an OpenAI implementation of the [image.ImageGeneration]
// interface. It also supports OpenAI-compatible providers (e.g. xAI Grok image)
// via [WithBaseURL].
package openai

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/model"
	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// StreamingOptions contains OpenAI-specific options for streaming image generation.
type StreamingOptions struct {
	// PartialImages specifies the number of partial images to receive during streaming (0-3).
	// If set to 0, only the final image will be received.
	// You may receive fewer partial images than requested if the full image is generated quickly.
	PartialImages int
}

// Options configures the OpenAI image generation client.
type Options struct {
	apiKey           string
	model            model.ImageGenerationModel
	timeout          *time.Duration
	baseURL          string
	extraHeaders     map[string]string
	streamingOptions StreamingOptions
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with OpenAI.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) {
		o.apiKey = apiKey
	}
}

// WithModel selects the image generation model.
func WithModel(m model.ImageGenerationModel) Option {
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

// WithBaseURL points the client at a custom OpenAI-compatible endpoint
// (e.g. "https://api.x.ai/v1" for xAI Grok image generation).
func WithBaseURL(baseURL string) Option {
	return func(o *Options) {
		o.baseURL = baseURL
	}
}

// WithExtraHeaders adds custom HTTP headers to every request.
func WithExtraHeaders(headers map[string]string) Option {
	return func(o *Options) {
		o.extraHeaders = headers
	}
}

// WithStreamingOptions configures streaming behaviour (e.g. partial-image count).
func WithStreamingOptions(opts StreamingOptions) Option {
	return func(o *Options) {
		o.streamingOptions = opts
	}
}

// Client implements [image.ImageGeneration] against the OpenAI image generation API.
type Client struct {
	options Options
	client  openaisdk.Client
}

// NewGeneration constructs an OpenAI image generation client. The returned
// [image.ImageGeneration] is wrapped with [image.WithTracing], so callers always
// get tracing spans and metrics.
func NewGeneration(opts ...Option) image.ImageGeneration {
	options := Options{
		streamingOptions: StreamingOptions{
			PartialImages: 2,
		},
	}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{
		option.WithAPIKey(options.apiKey),
	}
	if options.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(options.baseURL))
	}
	for k, v := range options.extraHeaders {
		clientOpts = append(clientOpts, option.WithHeader(k, v))
	}

	return image.WithTracing(&Client{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	})
}

// Model returns the configured image generation model.
func (c *Client) Model() model.ImageGenerationModel {
	return c.options.model
}

// GenerateImage performs a non-streaming image generation request.
func (c *Client) GenerateImage(
	ctx context.Context,
	prompt string,
	options ...image.GenerationOption,
) (*image.ImageGenerationResponse, error) {
	genOpts := image.GenerationOptions{
		Size:           c.options.model.DefaultSize,
		Quality:        c.options.model.DefaultQuality,
		ResponseFormat: "url",
		N:              1,
	}
	for _, opt := range options {
		opt(&genOpts)
	}

	params := openaisdk.ImageGenerateParams{
		Prompt: prompt,
		Model:  openaisdk.ImageModel(c.options.model.APIModel),
		N:      openaisdk.Int(int64(genOpts.N)),
	}

	if genOpts.ResponseFormat != "" &&
		c.options.model.APIModel != "gpt-image-1" &&
		c.options.model.APIModel != "gpt-image-1.5" &&
		c.options.model.APIModel != "gpt-image-1-mini" {
		params.ResponseFormat = openaisdk.ImageGenerateParamsResponseFormat(
			genOpts.ResponseFormat,
		)
	}

	if genOpts.Size != "" && len(c.options.model.SupportedSizes) > 0 {
		params.Size = openaisdk.ImageGenerateParamsSize(genOpts.Size)
	}

	if genOpts.Quality != "" && genOpts.Quality != "default" &&
		len(c.options.model.SupportedQualities) > 1 {
		params.Quality = openaisdk.ImageGenerateParamsQuality(genOpts.Quality)
	}

	if c.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *c.options.timeout)
		defer cancel()
	}

	response, err := c.client.Images.Generate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	results := make([]image.ImageGenerationResult, 0, len(response.Data))
	for _, img := range response.Data {
		result := image.ImageGenerationResult{
			RevisedPrompt: img.RevisedPrompt,
		}
		if img.URL != "" {
			result.ImageURL = img.URL
		}
		if img.B64JSON != "" {
			result.ImageBase64 = img.B64JSON
		}
		results = append(results, result)
	}

	return &image.ImageGenerationResponse{
		Images: results,
		Usage:  image.ImageGenerationUsage{PromptTokens: 0},
		Model:  c.options.model.APIModel,
	}, nil
}

// GenerateImageStreaming performs a streaming image generation request.
// Returns [image.ErrStreamingNotSupported] if the configured model does not
// support streaming.
func (c *Client) GenerateImageStreaming(
	ctx context.Context,
	prompt string,
	callback image.StreamCallback,
	options ...image.GenerationOption,
) error {
	if !c.options.model.SupportsStreaming {
		return image.ErrStreamingNotSupported
	}

	genOpts := image.GenerationOptions{
		Size:    c.options.model.DefaultSize,
		Quality: c.options.model.DefaultQuality,
		N:       1,
	}
	for _, opt := range options {
		opt(&genOpts)
	}

	params := openaisdk.ImageGenerateParams{
		Prompt: prompt,
		Model:  openaisdk.ImageModel(c.options.model.APIModel),
		N:      openaisdk.Int(int64(genOpts.N)),
		PartialImages: openaisdk.Int(
			int64(c.options.streamingOptions.PartialImages),
		),
	}

	if genOpts.Size != "" && len(c.options.model.SupportedSizes) > 0 {
		params.Size = openaisdk.ImageGenerateParamsSize(genOpts.Size)
	}

	if genOpts.Quality != "" && genOpts.Quality != "default" &&
		len(c.options.model.SupportedQualities) > 1 {
		params.Quality = openaisdk.ImageGenerateParamsQuality(genOpts.Quality)
	}

	if c.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *c.options.timeout)
		defer cancel()
	}

	stream := c.client.Images.GenerateStreaming(ctx, params)

	for stream.Next() {
		event := stream.Current()

		switch event.Type {
		case "image.partial_image":
			if err := callback(image.ImageStreamEvent{
				Type:              image.EventPartialImage,
				ImageBase64:       event.B64JSON,
				PartialImageIndex: int(event.PartialImageIndex),
				Size:              event.Size,
				Quality:           event.Quality,
			}); err != nil {
				return fmt.Errorf("callback error on partial image: %w", err)
			}

		case "image.completed":
			if err := callback(image.ImageStreamEvent{
				Type:        image.EventCompleted,
				ImageBase64: event.B64JSON,
				Size:        event.Size,
				Quality:     event.Quality,
			}); err != nil {
				return fmt.Errorf("callback error on completed image: %w", err)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("streaming error: %w", err)
	}
	return nil
}
