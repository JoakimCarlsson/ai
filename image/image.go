// Package image provides a unified interface for generating images from text prompts
// using various AI providers.
//
// This package defines the [Generation] interface and the data types that flow
// through it. Concrete vendor implementations live in subpackages (image/openai,
// image/gemini, image/xai); each subpackage exports its own NewGeneration
// constructor that returns a tracing-wrapped client implementing the interface.
//
// All vendor knobs (size, aspect ratio, quality, response format, style, seed,
// safety, ...) are configured at construction on the vendor's Options. The
// per-call surface is just the prompt — image generation is "configure once,
// prompt many" and vendor request bodies don't share enough common shape to
// support a portable per-call surface.
//
// Example usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/image"
//		imageopenai "github.com/joakimcarlsson/ai/image/openai"
//		"github.com/joakimcarlsson/ai/model"
//	)
//
//	client := imageopenai.NewGeneration(
//		imageopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
//		imageopenai.WithModel(model.OpenAIImageGenerationModels[model.DallE3]),
//		imageopenai.WithSize("1024x1024"),
//		imageopenai.WithQuality("hd"),
//		imageopenai.WithStyle("vivid"),
//	)
//
//	response, err := client.GenerateImage(ctx, "A serene mountain landscape at sunset")
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

// GenerationUsage tracks the resource consumption for image generation operations.
type GenerationUsage struct {
	// PromptTokens is the number of tokens in the input prompt.
	PromptTokens int64
}

// GenerationResult represents a single generated image with its metadata.
type GenerationResult struct {
	// ImageURL contains the URL to the generated image if the vendor returned a URL.
	ImageURL string
	// ImageBase64 contains the base64-encoded image data if the vendor returned bytes.
	ImageBase64 string
	// RevisedPrompt contains the prompt that was actually used to generate the image.
	RevisedPrompt string
}

// GenerationResponse contains the generated images and metadata from an image generation request.
type GenerationResponse struct {
	// Images contains the generated image results.
	Images []GenerationResult
	// Usage tracks resource consumption for this request.
	Usage GenerationUsage
	// Model identifies which image generation model was used.
	Model string
}

// StreamEventType identifies the type of streaming event during image generation.
type StreamEventType string

const (
	// EventPartialImage is emitted when a partial image is available during streaming.
	EventPartialImage StreamEventType = "partial_image"
	// EventCompleted is emitted when image generation is complete and the final image is available.
	EventCompleted StreamEventType = "completed"
)

// StreamEvent represents a streaming event during image generation.
type StreamEvent struct {
	// Type identifies the kind of streaming event.
	Type StreamEventType `json:"type"`
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
type StreamCallback func(StreamEvent) error

// ErrStreamingNotSupported is returned when streaming is requested but the model doesn't support it.
var ErrStreamingNotSupported = errors.New(
	"streaming not supported by this model",
)

// Generation defines the interface for generating images from text prompts.
type Generation interface {
	// GenerateImage creates one or more images from a text prompt. All vendor
	// configuration (size, quality, aspect ratio, ...) is set on the underlying
	// client at construction.
	GenerateImage(ctx context.Context, prompt string) (*GenerationResponse, error)

	// GenerateImageStreaming streams partial images during generation.
	// The callback is invoked for each partial image and the final completed image.
	// Returns ErrStreamingNotSupported if the model doesn't support streaming.
	GenerateImageStreaming(
		ctx context.Context,
		prompt string,
		callback StreamCallback,
	) error

	// Model returns the image generation model configuration being used.
	Model() model.ImageGenerationModel
}

// TracingAttrs are construction-time attributes vendor packages forward to the
// [WithTracing] wrapper so they appear on every span produced for the wrapped
// client.
type TracingAttrs struct{}

// WithTracing wraps a [Generation] client so every call records OpenTelemetry
// spans and metrics. The attrs are recorded as construction-time span attributes.
func WithTracing(inner Generation, attrs TracingAttrs) Generation {
	return &tracingClient{inner: inner, attrs: attrs}
}

type tracingClient struct {
	inner Generation
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
) (*GenerationResponse, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartImageSpan(
		ctx,
		m.APIModel,
		string(m.Provider),
		t.spanAttrs()...,
	)
	defer span.End()

	resp, err := t.inner.GenerateImage(ctx, prompt)
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

	err := t.inner.GenerateImageStreaming(ctx, prompt, callback)
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
