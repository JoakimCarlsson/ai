// Package azure provides an Azure OpenAI implementation of the
// [image.Generation] interface. Supports gpt-image-1.5 and gpt-image-2.
//
// Azure's image request/response semantics are identical to OpenAI's, so this
// package wraps [image/openai].Client and overrides only the SDK construction
// (custom auth + endpoint). It mirrors [llm/azure] for chat/completions.
package azure

import (
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joakimcarlsson/ai/image"
	imageopenai "github.com/joakimcarlsson/ai/image/openai"
	"github.com/joakimcarlsson/ai/model"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/option"
)

// Re-exported enums from [image/openai] so callers configure an Azure client
// without importing both packages.
type (
	// Size enumerates the image-dimension presets. See [image/openai.Size].
	Size = imageopenai.Size
	// Quality enumerates the per-image quality presets. See [image/openai.Quality].
	Quality = imageopenai.Quality
	// Background controls the generated image's background. See [image/openai.Background].
	Background = imageopenai.Background
	// Moderation sets the content-moderation strictness. See [image/openai.Moderation].
	Moderation = imageopenai.Moderation
	// OutputFormat selects the response image encoding. See [image/openai.OutputFormat].
	OutputFormat = imageopenai.OutputFormat
	// StreamingOptions configures streaming behaviour. See [image/openai.StreamingOptions].
	StreamingOptions = imageopenai.StreamingOptions
)

// Re-exported enum values from [image/openai], mirroring the aliased types above.
const (
	SizeAuto      = imageopenai.SizeAuto
	Size1024x1024 = imageopenai.Size1024x1024
	Size1024x1536 = imageopenai.Size1024x1536
	Size1536x1024 = imageopenai.Size1536x1024

	QualityLow    = imageopenai.QualityLow
	QualityMedium = imageopenai.QualityMedium
	QualityHigh   = imageopenai.QualityHigh
	QualityAuto   = imageopenai.QualityAuto

	BackgroundTransparent = imageopenai.BackgroundTransparent
	BackgroundOpaque      = imageopenai.BackgroundOpaque
	BackgroundAuto        = imageopenai.BackgroundAuto

	ModerationAuto = imageopenai.ModerationAuto
	ModerationLow  = imageopenai.ModerationLow

	OutputFormatPNG  = imageopenai.OutputFormatPNG
	OutputFormatJPEG = imageopenai.OutputFormatJPEG
	OutputFormatWebP = imageopenai.OutputFormatWebP
)

// Options configures the Azure OpenAI image generation client.
type Options struct {
	apiKey            string
	model             model.ImageGenerationModel
	timeout           *time.Duration
	endpoint          string
	apiVersion        string
	extraHeaders      map[string]string
	streamingOptions  *StreamingOptions
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

// WithAPIKey sets the API key (optional — Azure also supports
// DefaultAzureCredential when no key is provided).
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

// WithEndpoint sets the Azure OpenAI endpoint URL.
func WithEndpoint(endpoint string) Option {
	return func(o *Options) { o.endpoint = endpoint }
}

// WithAPIVersion sets the Azure OpenAI API version.
func WithAPIVersion(apiVersion string) Option {
	return func(o *Options) { o.apiVersion = apiVersion }
}

// WithExtraHeaders adds custom HTTP headers to every request.
func WithExtraHeaders(headers map[string]string) Option {
	return func(o *Options) { o.extraHeaders = headers }
}

// WithStreamingOptions configures streaming behaviour (e.g. partial-image count).
func WithStreamingOptions(opts StreamingOptions) Option {
	return func(o *Options) { o.streamingOptions = &opts }
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

// Client implements [image.Generation] against Azure OpenAI by delegating
// request handling to [image/openai].Client constructed with Azure-specific SDK
// options.
type Client struct {
	*imageopenai.Client
}

// NewGeneration constructs an Azure OpenAI image generation client. The returned
// [image.Generation] is wrapped with [image.WithTracing], so callers always get
// tracing spans and metrics — consistent with [image/openai.NewGeneration].
func NewGeneration(opts ...Option) image.Generation {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	imageOpts := passthroughOptions(options)

	if options.endpoint == "" || options.apiVersion == "" {
		if options.apiKey != "" {
			imageOpts = append(imageOpts, imageopenai.WithAPIKey(options.apiKey))
		}
		return imageopenai.NewGeneration(imageOpts...)
	}

	if strings.Contains(options.endpoint, "/openai/v1") {
		imageOpts = append(
			imageOpts,
			imageopenai.WithBaseURL(strings.TrimRight(options.endpoint, "/")),
		)
		if options.apiKey != "" {
			imageOpts = append(imageOpts, imageopenai.WithAPIKey(options.apiKey))
		}
		return imageopenai.NewGeneration(imageOpts...)
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(options.endpoint, options.apiVersion),
	}
	if options.apiKey != "" {
		reqOpts = append(reqOpts, azure.WithAPIKey(options.apiKey))
	} else if cred, err := azidentity.NewDefaultAzureCredential(nil); err == nil {
		reqOpts = append(reqOpts, azure.WithTokenCredential(cred))
	}

	bare := imageopenai.NewWithExistingClient(
		buildImageOptions(options),
		openaisdk.NewClient(reqOpts...),
	)
	return image.WithTracing(&Client{Client: bare}, image.TracingAttrs{})
}

// passthroughOptions builds the [image/openai] option list common to every
// branch (everything except auth/base-url, which each branch appends).
func passthroughOptions(o Options) []imageopenai.Option {
	imageOpts := []imageopenai.Option{
		imageopenai.WithModel(o.model),
	}
	if o.timeout != nil {
		imageOpts = append(imageOpts, imageopenai.WithTimeout(*o.timeout))
	}
	if o.extraHeaders != nil {
		imageOpts = append(imageOpts, imageopenai.WithExtraHeaders(o.extraHeaders))
	}
	if o.streamingOptions != nil {
		imageOpts = append(
			imageOpts,
			imageopenai.WithStreamingOptions(*o.streamingOptions),
		)
	}
	if o.n != nil {
		imageOpts = append(imageOpts, imageopenai.WithN(*o.n))
	}
	if o.size != "" {
		imageOpts = append(imageOpts, imageopenai.WithSize(o.size))
	}
	if o.quality != "" {
		imageOpts = append(imageOpts, imageopenai.WithQuality(o.quality))
	}
	if o.background != "" {
		imageOpts = append(imageOpts, imageopenai.WithBackground(o.background))
	}
	if o.moderation != "" {
		imageOpts = append(imageOpts, imageopenai.WithModeration(o.moderation))
	}
	if o.outputFormat != "" {
		imageOpts = append(imageOpts, imageopenai.WithOutputFormat(o.outputFormat))
	}
	if o.outputCompression != nil {
		imageOpts = append(
			imageOpts,
			imageopenai.WithOutputCompression(*o.outputCompression),
		)
	}
	if o.user != "" {
		imageOpts = append(imageOpts, imageopenai.WithUser(o.user))
	}
	return imageOpts
}

// buildImageOptions converts our Options to the wrapped image/openai package's
// Options. The image/openai package keeps its options struct fields unexported,
// so we go through the option-func ladder (mirroring llm/azure's
// buildOpenAIOptions). Auth/base-url are intentionally omitted — the classic
// Azure endpoint path supplies a pre-built client instead.
func buildImageOptions(o Options) imageopenai.Options {
	var dst imageopenai.Options
	for _, opt := range passthroughOptions(o) {
		opt(&dst)
	}
	return dst
}
