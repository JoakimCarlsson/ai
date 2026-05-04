// Package xai provides an xAI implementation of the [image.Generation] interface.
//
// xAI's image endpoint speaks the OpenAI request shape, so the underlying
// transport is openai-go pointed at xAI's base URL. xAI-specific request body
// fields (`aspect_ratio`, `resolution`, `user`) are injected via
// [option.WithJSONSet].
package xai

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/model"
	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// DefaultBaseURL is the canonical xAI API endpoint.
const DefaultBaseURL = "https://api.x.ai/v1"

// AspectRatio enumerates xAI's documented aspect-ratio values for
// grok-imagine-image and -pro. Stored as a typed string so callers can still
// pass a value outside the enum if xAI ships one before this list is updated.
type AspectRatio string

// Supported aspect-ratio values per https://docs.x.ai/docs/guides/image-generations.
const (
	AspectRatio1x1     AspectRatio = "1:1"
	AspectRatio16x9    AspectRatio = "16:9"
	AspectRatio9x16    AspectRatio = "9:16"
	AspectRatio4x3     AspectRatio = "4:3"
	AspectRatio3x4     AspectRatio = "3:4"
	AspectRatio3x2     AspectRatio = "3:2"
	AspectRatio2x3     AspectRatio = "2:3"
	AspectRatio2x1     AspectRatio = "2:1"
	AspectRatio1x2     AspectRatio = "1:2"
	AspectRatio19_5x9  AspectRatio = "19.5:9"
	AspectRatio9x19_5  AspectRatio = "9:19.5"
	AspectRatio20x9    AspectRatio = "20:9"
	AspectRatio9x20    AspectRatio = "9:20"
	AspectRatioAuto    AspectRatio = "auto"
)

// Resolution enumerates the output-resolution presets for Grok Imagine models.
type Resolution string

// Supported resolution values.
const (
	Resolution1K Resolution = "1k"
	Resolution2K Resolution = "2k"
)

// ResponseFormat selects how the response delivers the image.
type ResponseFormat string

// Supported response-format values.
const (
	ResponseFormatURL    ResponseFormat = "url"
	ResponseFormatBase64 ResponseFormat = "b64_json"
)

// Options configures the xAI image generation client.
type Options struct {
	apiKey         string
	model          model.ImageGenerationModel
	timeout        *time.Duration
	baseURL        string
	extraHeaders   map[string]string
	n              *int
	aspectRatio    AspectRatio
	resolution     Resolution
	responseFormat ResponseFormat
	user           string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with xAI.
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

// WithBaseURL overrides [DefaultBaseURL]. Useful for proxies or staging.
func WithBaseURL(baseURL string) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithExtraHeaders adds custom HTTP headers to every request.
func WithExtraHeaders(headers map[string]string) Option {
	return func(o *Options) { o.extraHeaders = headers }
}

// WithN sets how many images to generate per request (1–10).
func WithN(n int) Option {
	return func(o *Options) { o.n = &n }
}

// WithAspectRatio sets the aspect ratio. Supported by grok-imagine-image and
// grok-imagine-image-pro; see [AspectRatio] for valid values.
func WithAspectRatio(ratio AspectRatio) Option {
	return func(o *Options) { o.aspectRatio = ratio }
}

// WithResolution sets the output resolution for Grok Imagine models.
func WithResolution(res Resolution) Option {
	return func(o *Options) { o.resolution = res }
}

// WithResponseFormat selects how the response delivers the image.
func WithResponseFormat(format ResponseFormat) Option {
	return func(o *Options) { o.responseFormat = format }
}

// WithUser tags the request with an end-user identifier.
func WithUser(user string) Option {
	return func(o *Options) { o.user = user }
}

// Client implements [image.Generation] against the xAI image generation API.
type Client struct {
	options Options
	client  openaisdk.Client
}

// NewGeneration constructs an xAI image generation client. The returned
// [image.Generation] is wrapped with [image.WithTracing], so callers always
// get tracing spans and metrics.
func NewGeneration(opts ...Option) image.Generation {
	options := Options{baseURL: DefaultBaseURL}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{option.WithAPIKey(options.apiKey)}
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

// GenerateImage performs a non-streaming image generation request against xAI.
func (c *Client) GenerateImage(
	ctx context.Context,
	prompt string,
) (*image.GenerationResponse, error) {
	n := 1
	if c.options.n != nil {
		n = *c.options.n
	}

	params := openaisdk.ImageGenerateParams{
		Prompt: prompt,
		Model:  openaisdk.ImageModel(c.options.model.APIModel),
		N:      openaisdk.Int(int64(n)),
	}

	responseFormat := c.options.responseFormat
	if responseFormat == "" {
		responseFormat = ResponseFormatURL
	}
	params.ResponseFormat = openaisdk.ImageGenerateParamsResponseFormat(responseFormat)

	aspectRatio := c.options.aspectRatio
	if aspectRatio == "" {
		aspectRatio = AspectRatio(c.options.model.DefaultAspectRatio)
	}

	requestOpts := []option.RequestOption{}
	if aspectRatio != "" {
		requestOpts = append(requestOpts,
			option.WithJSONSet("aspect_ratio", string(aspectRatio)),
		)
	}
	if c.options.resolution != "" {
		requestOpts = append(requestOpts,
			option.WithJSONSet("resolution", string(c.options.resolution)),
		)
	}
	if c.options.user != "" {
		requestOpts = append(requestOpts,
			option.WithJSONSet("user", c.options.user),
		)
	}

	if c.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *c.options.timeout)
		defer cancel()
	}

	response, err := c.client.Images.Generate(ctx, params, requestOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	results := make([]image.GenerationResult, 0, len(response.Data))
	for _, img := range response.Data {
		result := image.GenerationResult{RevisedPrompt: img.RevisedPrompt}
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

// GenerateImageStreaming returns [image.ErrStreamingNotSupported]; xAI's image
// API does not document streaming.
func (c *Client) GenerateImageStreaming(
	_ context.Context,
	_ string,
	_ image.StreamCallback,
) error {
	return image.ErrStreamingNotSupported
}
