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

// Options configures the Gemini image generation client.
type Options struct {
	apiKey  string
	model   model.ImageGenerationModel
	timeout *time.Duration
	backend genai.Backend
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Gemini.
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

// WithBackend selects the Gemini backend (GeminiAPI or VertexAI).
func WithBackend(backend genai.Backend) Option {
	return func(o *Options) {
		o.backend = backend
	}
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

// GenerateImage performs a non-streaming image generation request.
func (c *Client) GenerateImage(
	ctx context.Context,
	prompt string,
	options ...image.GenerationOption,
) (*image.GenerationResponse, error) {
	genOpts := image.GenerationOptions{
		Size:           c.options.model.DefaultSize,
		Quality:        c.options.model.DefaultQuality,
		ResponseFormat: "b64_json",
		N:              1,
	}
	for _, opt := range options {
		opt(&genOpts)
	}

	config := &genai.GenerateImagesConfig{
		NumberOfImages: int32(genOpts.N),
	}
	if genOpts.Size != "" && genOpts.Size != "1:1" {
		config.AspectRatio = genOpts.Size
	}

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
	_ ...image.GenerationOption,
) error {
	return image.ErrStreamingNotSupported
}
