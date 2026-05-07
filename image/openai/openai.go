// Package openai provides an OpenAI implementation of the [image.Generation]
// interface. Supports gpt-image-1.5 and gpt-image-2.
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

// Size enumerates the image-dimension presets accepted by gpt-image-1.5 and
// gpt-image-2. Stored as a typed string so a caller can still pass a value
// outside the enum if OpenAI ships one before this list is updated.
type Size string

// Supported size values for gpt-image-1.5 / gpt-image-2.
const (
	SizeAuto      Size = "auto"
	Size1024x1024 Size = "1024x1024"
	Size1024x1536 Size = "1024x1536"
	Size1536x1024 Size = "1536x1024"
)

// Quality enumerates the per-image quality presets.
type Quality string

// Supported quality presets.
const (
	QualityLow    Quality = "low"
	QualityMedium Quality = "medium"
	QualityHigh   Quality = "high"
	QualityAuto   Quality = "auto"
)

// Background controls the generated image's background. Supported by
// gpt-image-1.5; gpt-image-2 silently rejects this field.
type Background string

// Supported background values.
const (
	BackgroundTransparent Background = "transparent"
	BackgroundOpaque      Background = "opaque"
	BackgroundAuto        Background = "auto"
)

// Moderation sets the content-moderation strictness.
type Moderation string

// Supported moderation values.
const (
	ModerationAuto Moderation = "auto"
	ModerationLow  Moderation = "low"
)

// OutputFormat selects the response image encoding.
type OutputFormat string

// Supported output formats.
const (
	OutputFormatPNG  OutputFormat = "png"
	OutputFormatJPEG OutputFormat = "jpeg"
	OutputFormatWebP OutputFormat = "webp"
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
	apiKey            string
	model             model.ImageGenerationModel
	timeout           *time.Duration
	baseURL           string
	extraHeaders      map[string]string
	streamingOptions  StreamingOptions
	n                 *int
	size              Size
	quality           Quality
	background        Background
	moderation        Moderation
	outputFormat      OutputFormat
	outputCompression *int
	user              string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with OpenAI.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the image generation model.
func WithModel(m model.ImageGenerationModel) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithBaseURL points the client at a custom OpenAI-compatible endpoint.
func WithBaseURL(baseURL string) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithExtraHeaders adds custom HTTP headers to every request.
func WithExtraHeaders(headers map[string]string) Option {
	return func(o *Options) { o.extraHeaders = headers }
}

// WithStreamingOptions configures streaming behaviour (e.g. partial-image count).
func WithStreamingOptions(opts StreamingOptions) Option {
	return func(o *Options) { o.streamingOptions = opts }
}

// WithN sets how many images to generate per request (1–10).
func WithN(n int) Option {
	return func(o *Options) { o.n = &n }
}

// WithSize sets the image dimensions. See [Size] for valid values.
func WithSize(s Size) Option {
	return func(o *Options) { o.size = s }
}

// WithQuality sets the per-image quality preset. See [Quality] for valid values.
func WithQuality(q Quality) Option {
	return func(o *Options) { o.quality = q }
}

// WithBackground requests transparent/opaque/auto. Supported by gpt-image-1.5;
// gpt-image-2 silently rejects this field.
func WithBackground(b Background) Option {
	return func(o *Options) { o.background = b }
}

// WithModeration sets the content-moderation strictness.
func WithModeration(m Moderation) Option {
	return func(o *Options) { o.moderation = m }
}

// WithOutputFormat selects the response image encoding.
func WithOutputFormat(f OutputFormat) Option {
	return func(o *Options) { o.outputFormat = f }
}

// WithOutputCompression sets jpeg/webp compression quality (0–100).
func WithOutputCompression(quality int) Option {
	return func(o *Options) { o.outputCompression = &quality }
}

// WithUser tags the request with an end-user identifier for abuse-monitoring.
func WithUser(user string) Option {
	return func(o *Options) { o.user = user }
}

// Client implements [image.Generation] against the OpenAI image generation API.
type Client struct {
	options Options
	client  openaisdk.Client
}

// NewGeneration constructs an OpenAI image generation client. The returned
// [image.Generation] is wrapped with [image.WithTracing], so callers always
// get tracing spans and metrics.
func NewGeneration(opts ...Option) image.Generation {
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
	}, image.TracingAttrs{})
}

// Model returns the configured image generation model.
func (c *Client) Model() model.ImageGenerationModel {
	return c.options.model
}

func (c *Client) buildParams(prompt string) openaisdk.ImageGenerateParams {
	apiModel := c.options.model.APIModel
	params := openaisdk.ImageGenerateParams{
		Prompt: prompt,
		Model:  openaisdk.ImageModel(apiModel),
	}

	n := 1
	if c.options.n != nil {
		n = *c.options.n
	}
	params.N = openaisdk.Int(int64(n))

	size := c.options.size
	if size == "" {
		size = Size(c.options.model.DefaultSize)
	}
	if size != "" {
		params.Size = openaisdk.ImageGenerateParamsSize(size)
	}

	quality := c.options.quality
	if quality == "" {
		quality = Quality(c.options.model.DefaultQuality)
	}
	if quality != "" {
		params.Quality = openaisdk.ImageGenerateParamsQuality(quality)
	}

	if c.options.background != "" && apiModel != "gpt-image-2" {
		params.Background = openaisdk.ImageGenerateParamsBackground(
			c.options.background,
		)
	}
	if c.options.moderation != "" {
		params.Moderation = openaisdk.ImageGenerateParamsModeration(
			c.options.moderation,
		)
	}
	if c.options.outputFormat != "" {
		params.OutputFormat = openaisdk.ImageGenerateParamsOutputFormat(
			c.options.outputFormat,
		)
	}
	if c.options.outputCompression != nil {
		params.OutputCompression = openaisdk.Int(
			int64(*c.options.outputCompression),
		)
	}
	if c.options.user != "" {
		params.User = openaisdk.String(c.options.user)
	}

	return params
}

// GenerateImage performs a non-streaming image generation request.
func (c *Client) GenerateImage(
	ctx context.Context,
	prompt string,
) (*image.GenerationResponse, error) {
	params := c.buildParams(prompt)

	if c.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *c.options.timeout)
		defer cancel()
	}

	response, err := c.client.Images.Generate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	results := make([]image.GenerationResult, 0, len(response.Data))
	for _, img := range response.Data {
		result := image.GenerationResult{
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

	return &image.GenerationResponse{
		Images: results,
		Usage:  image.GenerationUsage{PromptTokens: 0},
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
) error {
	if !c.options.model.SupportsStreaming {
		return image.ErrStreamingNotSupported
	}

	params := c.buildParams(prompt)
	params.PartialImages = openaisdk.Int(
		int64(c.options.streamingOptions.PartialImages),
	)

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
			if err := callback(image.StreamEvent{
				Type:              image.EventPartialImage,
				ImageBase64:       event.B64JSON,
				PartialImageIndex: int(event.PartialImageIndex),
				Size:              event.Size,
				Quality:           event.Quality,
			}); err != nil {
				return fmt.Errorf("callback error on partial image: %w", err)
			}

		case "image.completed":
			if err := callback(image.StreamEvent{
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
