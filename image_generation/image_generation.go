// Package image_generation provides a unified interface for generating images from text prompts
// using various AI providers.
//
// This package abstracts the differences between image generation providers like OpenAI, xAI, and Gemini,
// offering a consistent API for creating images from natural language descriptions.
//
// Key features include:
//   - Text-to-image generation from prompts
//   - Support for multiple output formats (URL, base64)
//   - Configurable image quality and size (provider-dependent)
//   - Helper functions for downloading and decoding images
//   - Cost tracking per generated image
//
// Example usage:
//
//	client, err := image_generation.NewImageGeneration(model.ProviderXAI,
//		image_generation.WithAPIKey("your-api-key"),
//		image_generation.WithModel(model.XAIImageGenerationModels[model.XAIGrok2Image]),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	response, err := client.GenerateImage(ctx, "A serene mountain landscape at sunset",
//		image_generation.WithResponseFormat("b64_json"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	imageData, _ := image_generation.DecodeBase64Image(response.Images[0].ImageBase64)
//	os.WriteFile("image.png", imageData, 0644)
package image_generation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// ImageGenerationUsage tracks the resource consumption for image generation operations.
type ImageGenerationUsage struct {
	// PromptTokens is the number of tokens in the input prompt.
	PromptTokens int64
}

// ImageGenerationResult represents a single generated image with its metadata.
type ImageGenerationResult struct {
	// ImageURL contains the URL to the generated image if ResponseFormat was "url".
	ImageURL string
	// ImageBase64 contains the base64-encoded image data if ResponseFormat was "b64_json".
	ImageBase64 string
	// RevisedPrompt contains the prompt that was actually used to generate the image.
	RevisedPrompt string
}

// ImageGenerationResponse contains the generated images and metadata from an image generation request.
type ImageGenerationResponse struct {
	// Images contains the generated image results.
	Images []ImageGenerationResult
	// Usage tracks resource consumption for this request.
	Usage ImageGenerationUsage
	// Model identifies which image generation model was used.
	Model string
}

// ImageStreamEventType identifies the type of streaming event during image generation.
type ImageStreamEventType string

const (
	// EventPartialImage is emitted when a partial image is available during streaming.
	EventPartialImage ImageStreamEventType = "partial_image"
	// EventCompleted is emitted when image generation is complete and the final image is available.
	EventCompleted ImageStreamEventType = "completed"
)

// ImageStreamEvent represents a streaming event during image generation.
type ImageStreamEvent struct {
	// Type identifies the kind of streaming event.
	Type ImageStreamEventType `json:"type"`
	// ImageBase64 contains the base64-encoded image data.
	ImageBase64 string `json:"image_base64"`
	// PartialImageIndex is the 0-based index of the partial image (only for partial_image events).
	PartialImageIndex int `json:"partial_image_index,omitempty"`
	// Size is the dimensions of the image.
	Size string `json:"size,omitempty"`
	// Quality is the quality setting of the image.
	Quality string `json:"quality,omitempty"`
}

// StreamCallback is called for each streaming event during image generation.
type StreamCallback func(ImageStreamEvent) error

// ErrStreamingNotSupported is returned when streaming is requested but the model doesn't support it.
var ErrStreamingNotSupported = errors.New("streaming not supported by this model")

// ImageGeneration defines the interface for generating images from text prompts.
type ImageGeneration interface {
	// GenerateImage creates one or more images from a text prompt.
	// The optional GenerationOption parameters can customize the generation (size, quality, format, etc.).
	GenerateImage(
		ctx context.Context,
		prompt string,
		options ...GenerationOption,
	) (*ImageGenerationResponse, error)

	// GenerateImageStreaming streams partial images during generation.
	// The callback is invoked for each partial image and the final completed image.
	// Returns ErrStreamingNotSupported if the model doesn't support streaming.
	GenerateImageStreaming(
		ctx context.Context,
		prompt string,
		callback StreamCallback,
		options ...GenerationOption,
	) error

	// Model returns the image generation model configuration being used.
	Model() model.ImageGenerationModel
}

type imageGenerationClientOptions struct {
	apiKey  string
	model   model.ImageGenerationModel
	timeout *time.Duration

	openaiOptions []OpenAIOption
	geminiOptions []GeminiOption
}

type ImageGenerationClientOption func(*imageGenerationClientOptions)

type ImageGenerationClient interface {
	generate(
		ctx context.Context,
		prompt string,
		options ...GenerationOption,
	) (*ImageGenerationResponse, error)
}

// StreamingImageGenerationClient is an optional interface for clients that support streaming.
type StreamingImageGenerationClient interface {
	generateStreaming(
		ctx context.Context,
		prompt string,
		callback StreamCallback,
		options ...GenerationOption,
	) error
}

type baseImageGeneration[C ImageGenerationClient] struct {
	options imageGenerationClientOptions
	client  C
}

// NewImageGeneration creates a new image generation client for the specified provider.
// Supported providers include OpenAI, xAI, and Gemini. Use WithModel() to specify the image generation model
// and WithAPIKey() for authentication.
func NewImageGeneration(
	provider model.ModelProvider,
	opts ...ImageGenerationClientOption,
) (ImageGeneration, error) {
	clientOptions := imageGenerationClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderOpenAI:
		return &baseImageGeneration[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case model.ProviderXAI:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://api.x.ai/v1"),
		)
		return &baseImageGeneration[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case model.ProviderGemini:
		return &baseImageGeneration[GeminiClient]{
			options: clientOptions,
			client:  newGeminiClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf(
		"image generation provider not supported: %s",
		provider,
	)
}

func (i *baseImageGeneration[C]) GenerateImage(
	ctx context.Context,
	prompt string,
	options ...GenerationOption,
) (*ImageGenerationResponse, error) {
	return i.client.generate(ctx, prompt, options...)
}

func (i *baseImageGeneration[C]) GenerateImageStreaming(
	ctx context.Context,
	prompt string,
	callback StreamCallback,
	options ...GenerationOption,
) error {
	// Check if the client supports streaming
	if streamingClient, ok := any(i.client).(StreamingImageGenerationClient); ok {
		return streamingClient.generateStreaming(ctx, prompt, callback, options...)
	}
	return ErrStreamingNotSupported
}

func (i *baseImageGeneration[C]) Model() model.ImageGenerationModel {
	return i.options.model
}

// WithAPIKey sets the API key for authentication with the image generation provider.
func WithAPIKey(apiKey string) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.apiKey = apiKey
	}
}

// WithModel specifies which image generation model to use for creating images.
func WithModel(model model.ImageGenerationModel) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.model = model
	}
}

// WithTimeout sets the maximum duration to wait for image generation requests to complete.
func WithTimeout(timeout time.Duration) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.timeout = &timeout
	}
}

// WithOpenAIOptions applies OpenAI-specific configuration options.
// Also used for xAI since it uses OpenAI-compatible API.
func WithOpenAIOptions(
	openaiOptions ...OpenAIOption,
) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.openaiOptions = openaiOptions
	}
}

// WithGeminiOptions applies Gemini-specific configuration options.
func WithGeminiOptions(
	geminiOptions ...GeminiOption,
) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.geminiOptions = geminiOptions
	}
}

// GenerationOptions contains parameters for customizing image generation requests.
type GenerationOptions struct {
	// Size specifies the dimensions of the generated image (e.g., "1024x1024").
	// Not supported by all providers.
	Size string
	// Quality controls the quality level of the generated image (e.g., "standard", "hd").
	Quality string
	// ResponseFormat specifies the format of the response ("url" or "b64_json").
	ResponseFormat string
	// N is the number of images to generate from the prompt.
	N int
}

// GenerationOption is a function that configures GenerationOptions.
type GenerationOption func(*GenerationOptions)

// WithSize sets the dimensions of the generated image.
// Not all providers support this option. Format is typically "WIDTHxHEIGHT" (e.g., "1024x1024").
func WithSize(size string) GenerationOption {
	return func(options *GenerationOptions) {
		options.Size = size
	}
}

// WithQuality sets the quality level of the generated image.
// Common values are "standard" and "hd" (high definition).
func WithQuality(quality string) GenerationOption {
	return func(options *GenerationOptions) {
		options.Quality = quality
	}
}

// WithResponseFormat specifies how the generated image should be returned.
// Valid values are "url" (returns a URL to the image) or "b64_json" (returns base64-encoded image data).
func WithResponseFormat(format string) GenerationOption {
	return func(options *GenerationOptions) {
		options.ResponseFormat = format
	}
}

// WithN sets the number of images to generate from the prompt.
// Most providers charge per image generated.
func WithN(n int) GenerationOption {
	return func(options *GenerationOptions) {
		options.N = n
	}
}
