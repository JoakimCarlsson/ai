package image_generation

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client  *genai.Client
	options imageGenerationClientOptions
}

type geminiOptions struct {
	backend genai.Backend
}

// GeminiOption is a function that configures Gemini-specific options.
type GeminiOption func(*geminiOptions)

// WithGeminiBackend sets the backend for the Gemini API (GeminiAPI or VertexAI).
func WithGeminiBackend(backend genai.Backend) GeminiOption {
	return func(options *geminiOptions) {
		options.backend = backend
	}
}

func newGeminiClient(opts imageGenerationClientOptions) GeminiClient {
	geminiOpts := geminiOptions{
		backend: genai.BackendGeminiAPI,
	}

	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey:  opts.apiKey,
			Backend: geminiOpts.backend,
		},
	)
	if err != nil {
		return GeminiClient{}
	}

	return GeminiClient{
		client:  client,
		options: opts,
	}
}

func (g GeminiClient) generate(
	ctx context.Context,
	prompt string,
	options ...GenerationOption,
) (*ImageGenerationResponse, error) {
	genOpts := GenerationOptions{
		Size:           g.options.model.DefaultSize,
		Quality:        g.options.model.DefaultQuality,
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

	if g.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *g.options.timeout)
		defer cancel()
	}

	response, err := g.client.Models.GenerateImages(
		ctx,
		g.options.model.APIModel,
		prompt,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	results := make([]ImageGenerationResult, 0, len(response.GeneratedImages))
	for _, img := range response.GeneratedImages {
		b64Image := base64.StdEncoding.EncodeToString(img.Image.ImageBytes)
		result := ImageGenerationResult{
			ImageBase64: b64Image,
		}
		results = append(results, result)
	}

	return &ImageGenerationResponse{
		Images: results,
		Usage: ImageGenerationUsage{
			PromptTokens: 0,
		},
		Model: g.options.model.APIModel,
	}, nil
}
