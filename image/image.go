// Package image provides a unified interface for generating images from text prompts
// using various AI providers.
//
// This package defines the [ImageGeneration] interface and the data types that flow
// through it. Concrete vendor implementations live in subpackages (image/openai,
// image/gemini); each subpackage exports its own NewGeneration constructor that
// returns a tracing-wrapped client implementing the interface.
//
// Example usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/image"
//		imageopenai "github.com/joakimcarlsson/ai/image/openai"
//	)
//
//	client := imageopenai.NewGeneration(
//		imageopenai.WithAPIKey("your-api-key"),
//		imageopenai.WithModel(model.OpenAIImageModels[model.DallE3]),
//	)
//
//	response, err := client.GenerateImage(ctx, "A serene mountain landscape at sunset",
//		image.WithResponseFormat("b64_json"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	imageData, _ := image.DecodeBase64Image(response.Images[0].ImageBase64)
//	os.WriteFile("image.png", imageData, 0644)
package image

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tracing"
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
var ErrStreamingNotSupported = errors.New(
	"streaming not supported by this model",
)

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

// TracingAttrs are construction-time attributes vendor packages forward to the
// [WithTracing] wrapper so they appear on every span produced for the wrapped
// client.
type TracingAttrs struct{}

// WithTracing wraps an ImageGeneration client so every call records OpenTelemetry
// spans and metrics. The attrs are recorded as construction-time span attributes.
func WithTracing(inner ImageGeneration, attrs TracingAttrs) ImageGeneration {
	return &tracingClient{inner: inner, attrs: attrs}
}

type tracingClient struct {
	inner ImageGeneration
	attrs TracingAttrs
}

func (t *tracingClient) Model() model.ImageGenerationModel {
	return t.inner.Model()
}

func (t *tracingClient) spanAttrs() []tracing.Attr {
	return nil
}

func (t *tracingClient) GenerateImage(
	ctx context.Context,
	prompt string,
	options ...GenerationOption,
) (*ImageGenerationResponse, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartImageSpan(
		ctx,
		m.APIModel,
		string(m.Provider),
		t.spanAttrs()...,
	)
	defer span.End()

	resp, err := t.inner.GenerateImage(ctx, prompt, options...)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx,
			"generate_image",
			m.APIModel,
			string(m.Provider),
			time.Since(start),
			0,
			0,
			err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageInputTokens.Int64(int64(resp.Usage.PromptTokens)),
		tracing.AttrResultCount.Int(len(resp.Images)),
	)
	tracing.RecordMetrics(
		ctx,
		"generate_image",
		m.APIModel,
		string(m.Provider),
		time.Since(start),
		int64(resp.Usage.PromptTokens),
		0,
		nil,
	)
	return resp, nil
}

func (t *tracingClient) GenerateImageStreaming(
	ctx context.Context,
	prompt string,
	callback StreamCallback,
	options ...GenerationOption,
) error {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartImageSpan(
		ctx,
		m.APIModel,
		string(m.Provider),
		t.spanAttrs()...,
	)
	defer span.End()

	err := t.inner.GenerateImageStreaming(ctx, prompt, callback, options...)
	tracing.RecordMetrics(
		ctx,
		"generate_image",
		m.APIModel,
		string(m.Provider),
		time.Since(start),
		0,
		0,
		err,
	)
	if err != nil {
		tracing.SetError(span, err)
	}
	return err
}

// DownloadImage downloads an image from a URL and returns its binary data.
// This is a helper function for processing image generation responses that return URLs.
func DownloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"failed to download image: status code %d",
			resp.StatusCode,
		)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return data, nil
}

// DecodeBase64Image decodes a base64-encoded image string into binary data.
// This is a helper function for processing image generation responses with base64 format.
func DecodeBase64Image(b64 string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %w", err)
	}
	return data, nil
}
