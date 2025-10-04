package image_generation

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

type ImageGenerationUsage struct {
	PromptTokens int64
}

type ImageGenerationResult struct {
	ImageURL    string
	ImageBase64 string
	RevisedPrompt string
}

type ImageGenerationResponse struct {
	Images []ImageGenerationResult
	Usage  ImageGenerationUsage
	Model  string
}

type ImageGeneration interface {
	GenerateImage(
		ctx context.Context,
		prompt string,
		options ...GenerationOption,
	) (*ImageGenerationResponse, error)

	Model() model.ImageGenerationModel
}

type imageGenerationClientOptions struct {
	apiKey  string
	model   model.ImageGenerationModel
	timeout *time.Duration

	xaiOptions []XAIOption
}

type ImageGenerationClientOption func(*imageGenerationClientOptions)

type ImageGenerationClient interface {
	generate(
		ctx context.Context,
		prompt string,
		options ...GenerationOption,
	) (*ImageGenerationResponse, error)
}

type baseImageGeneration[C ImageGenerationClient] struct {
	options imageGenerationClientOptions
	client  C
}

func NewImageGeneration(
	provider model.ModelProvider,
	opts ...ImageGenerationClientOption,
) (ImageGeneration, error) {
	clientOptions := imageGenerationClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderXAI:
		return &baseImageGeneration[XAIClient]{
			options: clientOptions,
			client:  newXAIClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf("image generation provider not supported: %s", provider)
}

func (i *baseImageGeneration[C]) GenerateImage(
	ctx context.Context,
	prompt string,
	options ...GenerationOption,
) (*ImageGenerationResponse, error) {
	return i.client.generate(ctx, prompt, options...)
}

func (i *baseImageGeneration[C]) Model() model.ImageGenerationModel {
	return i.options.model
}

func WithAPIKey(apiKey string) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.apiKey = apiKey
	}
}

func WithModel(model model.ImageGenerationModel) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.model = model
	}
}

func WithTimeout(timeout time.Duration) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.timeout = &timeout
	}
}

func WithXAIOptions(xaiOptions ...XAIOption) ImageGenerationClientOption {
	return func(options *imageGenerationClientOptions) {
		options.xaiOptions = xaiOptions
	}
}

type GenerationOptions struct {
	Size           string
	Quality        string
	ResponseFormat string
	N              int
}

type GenerationOption func(*GenerationOptions)

func WithSize(size string) GenerationOption {
	return func(options *GenerationOptions) {
		options.Size = size
	}
}

func WithQuality(quality string) GenerationOption {
	return func(options *GenerationOptions) {
		options.Quality = quality
	}
}

func WithResponseFormat(format string) GenerationOption {
	return func(options *GenerationOptions) {
		options.ResponseFormat = format
	}
}

func WithN(n int) GenerationOption {
	return func(options *GenerationOptions) {
		options.N = n
	}
}
