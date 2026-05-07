// Package gemini provides a Google Gemini implementation of the
// [image.Generation] interface.
package gemini

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/model"
	"google.golang.org/genai"
)

// AspectRatio enumerates aspect-ratio values across the Imagen 4 family and
// Gemini Image variants. Imagen 4 supports the first 5; Gemini 2.5 Flash Image
// and Gemini 3 Pro Image accept the full 10. Stored as a typed string so a
// caller can still pass a value outside the enum if Google ships one before
// this list is updated.
type AspectRatio string

// Aspect-ratio values per the Imagen and Gemini Image documentation.
const (
	AspectRatio1x1  AspectRatio = "1:1"
	AspectRatio3x4  AspectRatio = "3:4"
	AspectRatio4x3  AspectRatio = "4:3"
	AspectRatio9x16 AspectRatio = "9:16"
	AspectRatio16x9 AspectRatio = "16:9"
	// Gemini Image only:
	AspectRatio2x3  AspectRatio = "2:3"
	AspectRatio3x2  AspectRatio = "3:2"
	AspectRatio4x5  AspectRatio = "4:5"
	AspectRatio5x4  AspectRatio = "5:4"
	AspectRatio21x9 AspectRatio = "21:9"
)

// ImageSize enumerates the largest-dimension presets. Imagen 4 std/ultra accept
// 1K and 2K; Gemini 3 Pro Image additionally accepts 4K. Imagen 4 Fast and
// Gemini 2.5 Flash Image have a fixed dimension and reject this field.
type ImageSize string

// Supported image-size presets.
const (
	ImageSize1K ImageSize = "1K"
	ImageSize2K ImageSize = "2K"
	ImageSize4K ImageSize = "4K"
)

// OutputMIMEType selects the response image MIME type. Imagen-only.
type OutputMIMEType string

// Supported output-MIME-type values.
const (
	OutputMIMETypePNG  OutputMIMEType = "image/png"
	OutputMIMETypeJPEG OutputMIMEType = "image/jpeg"
)

// Options configures the Gemini image generation client.
type Options struct {
	apiKey                   string
	model                    model.ImageGenerationModel
	timeout                  *time.Duration
	backend                  genai.Backend
	n                        *int32
	aspectRatio              AspectRatio
	negativePrompt           string
	seed                     *int32
	personGeneration         *genai.PersonGeneration
	safetyFilterLevel        *genai.SafetyFilterLevel
	language                 *genai.ImagePromptLanguage
	enhancePrompt            *bool
	imageSize                ImageSize
	includeRAIReason         *bool
	outputMIMEType           OutputMIMEType
	outputCompressionQuality *int32
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Gemini.
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

// WithBackend selects the Gemini backend (GeminiAPI or VertexAI).
func WithBackend(backend genai.Backend) Option {
	return func(o *Options) { o.backend = backend }
}

// WithN sets how many images to generate (Imagen accepts 1–4).
func WithN(n int32) Option {
	return func(o *Options) { o.n = &n }
}

// WithAspectRatio sets the aspect ratio. See [AspectRatio] for valid values
// (Imagen accepts the first 5; Gemini Image accepts all 10).
func WithAspectRatio(ratio AspectRatio) Option {
	return func(o *Options) { o.aspectRatio = ratio }
}

// WithNegativePrompt describes content the model should avoid. Imagen-only.
func WithNegativePrompt(prompt string) Option {
	return func(o *Options) { o.negativePrompt = prompt }
}

// WithSeed sets the random seed for reproducible generation. Imagen-only.
// Note: Imagen requires AddWatermark=false to honour the seed.
func WithSeed(seed int32) Option {
	return func(o *Options) { o.seed = &seed }
}

// WithPersonGeneration configures whether and how people may appear.
// Pass one of [genai.PersonGenerationDontAllow], [genai.PersonGenerationAllowAdult],
// or [genai.PersonGenerationAllowAll].
func WithPersonGeneration(p genai.PersonGeneration) Option {
	return func(o *Options) { o.personGeneration = &p }
}

// WithSafetyFilterLevel sets the safety-filter strictness. Imagen-only.
// Pass one of the [genai.SafetyFilterLevel] constants.
func WithSafetyFilterLevel(level genai.SafetyFilterLevel) Option {
	return func(o *Options) { o.safetyFilterLevel = &level }
}

// WithLanguage declares the prompt language. Imagen-only.
// Pass one of the [genai.ImagePromptLanguage] constants.
func WithLanguage(lang genai.ImagePromptLanguage) Option {
	return func(o *Options) { o.language = &lang }
}

// WithEnhancePrompt toggles Imagen's prompt-rewriting logic. Imagen-only.
func WithEnhancePrompt(enhance bool) Option {
	return func(o *Options) { o.enhancePrompt = &enhance }
}

// WithImageSize sets the largest-dimension target. See [ImageSize] for valid
// values. Imagen 4 std/ultra accept 1K/2K; gemini-3-pro-image-preview adds 4K.
// Imagen 4 Fast and gemini-2.5-flash-image have a fixed dimension and reject
// this field.
func WithImageSize(size ImageSize) Option {
	return func(o *Options) { o.imageSize = size }
}

// WithIncludeRAIReason includes the Responsible AI block reason in the
// response when an image is filtered out. Imagen-only.
func WithIncludeRAIReason(include bool) Option {
	return func(o *Options) { o.includeRAIReason = &include }
}

// WithOutputMIMEType sets the response MIME type. Imagen-only. See
// [OutputMIMEType] for valid values.
func WithOutputMIMEType(mime OutputMIMEType) Option {
	return func(o *Options) { o.outputMIMEType = mime }
}

// WithOutputCompressionQuality sets jpeg compression quality (0–100). Imagen
// jpeg only.
func WithOutputCompressionQuality(quality int32) Option {
	return func(o *Options) { o.outputCompressionQuality = &quality }
}

// Client implements [image.Generation] against the Google Gemini API.
type Client struct {
	options Options
	client  *genai.Client
}

// NewGeneration constructs a Gemini image generation client. The returned
// [image.Generation] is wrapped with [image.WithTracing], so callers always
// get tracing spans and metrics.
func NewGeneration(opts ...Option) image.Generation {
	options := Options{
		backend: genai.BackendGeminiAPI,
	}
	for _, o := range opts {
		o(&options)
	}

	client, _ := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey:  options.apiKey,
			Backend: options.backend,
		},
	)

	return image.WithTracing(&Client{
		options: options,
		client:  client,
	}, image.TracingAttrs{})
}

// Model returns the configured image generation model.
func (c *Client) Model() model.ImageGenerationModel {
	return c.options.model
}

func (c *Client) buildConfig() *genai.GenerateImagesConfig {
	config := &genai.GenerateImagesConfig{}

	n := int32(1)
	if c.options.n != nil {
		n = *c.options.n
	}
	config.NumberOfImages = n

	aspect := c.options.aspectRatio
	if aspect == "" {
		aspect = AspectRatio(c.options.model.DefaultAspectRatio)
	}
	if aspect != "" && aspect != AspectRatio1x1 {
		config.AspectRatio = string(aspect)
	}

	if c.options.negativePrompt != "" {
		config.NegativePrompt = c.options.negativePrompt
	}
	if c.options.seed != nil {
		config.Seed = c.options.seed
	}
	if c.options.personGeneration != nil {
		config.PersonGeneration = *c.options.personGeneration
	}
	if c.options.safetyFilterLevel != nil {
		config.SafetyFilterLevel = *c.options.safetyFilterLevel
	}
	if c.options.language != nil {
		config.Language = *c.options.language
	}
	if c.options.enhancePrompt != nil {
		config.EnhancePrompt = *c.options.enhancePrompt
	}
	if c.options.imageSize != "" {
		config.ImageSize = string(c.options.imageSize)
	}
	if c.options.includeRAIReason != nil {
		config.IncludeRAIReason = *c.options.includeRAIReason
	}
	if c.options.outputMIMEType != "" {
		config.OutputMIMEType = string(c.options.outputMIMEType)
	}
	if c.options.outputCompressionQuality != nil {
		config.OutputCompressionQuality = c.options.outputCompressionQuality
	}

	return config
}

// GenerateImage performs a non-streaming image generation request.
func (c *Client) GenerateImage(
	ctx context.Context,
	prompt string,
) (*image.GenerationResponse, error) {
	config := c.buildConfig()

	if c.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *c.options.timeout)
		defer cancel()
	}

	response, err := c.client.Models.GenerateImages(
		ctx,
		c.options.model.APIModel,
		prompt,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	results := make(
		[]image.GenerationResult,
		0,
		len(response.GeneratedImages),
	)
	for _, img := range response.GeneratedImages {
		results = append(results, image.GenerationResult{
			ImageBase64: base64.StdEncoding.EncodeToString(
				img.Image.ImageBytes,
			),
		})
	}

	return &image.GenerationResponse{
		Images: results,
		Usage:  image.GenerationUsage{PromptTokens: 0},
		Model:  c.options.model.APIModel,
	}, nil
}

// GenerateImageStreaming returns [image.ErrStreamingNotSupported]; the Gemini
// API does not currently expose streaming image generation.
func (c *Client) GenerateImageStreaming(
	_ context.Context,
	_ string,
	_ image.StreamCallback,
) error {
	return image.ErrStreamingNotSupported
}
