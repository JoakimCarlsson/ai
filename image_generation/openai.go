package image_generation

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIClient struct {
	client  openai.Client
	options imageGenerationClientOptions
}

type openaiOptions struct {
	baseURL      string
	extraHeaders map[string]string
}

// OpenAIOption is a function that configures OpenAI-specific options.
type OpenAIOption func(*openaiOptions)

// WithOpenAIBaseURL sets a custom base URL for the OpenAI API endpoint.
func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(options *openaiOptions) {
		options.baseURL = baseURL
	}
}

// WithOpenAIExtraHeaders adds custom HTTP headers to OpenAI API requests.
func WithOpenAIExtraHeaders(headers map[string]string) OpenAIOption {
	return func(options *openaiOptions) {
		options.extraHeaders = headers
	}
}

func newOpenAIClient(opts imageGenerationClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{
		baseURL:      "",
		extraHeaders: make(map[string]string),
	}

	for _, o := range opts.openaiOptions {
		o(&openaiOpts)
	}

	clientOpts := []option.RequestOption{
		option.WithAPIKey(opts.apiKey),
	}

	if openaiOpts.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(openaiOpts.baseURL))
	}

	for k, v := range openaiOpts.extraHeaders {
		clientOpts = append(clientOpts, option.WithHeader(k, v))
	}

	client := openai.NewClient(clientOpts...)

	return OpenAIClient{
		client:  client,
		options: opts,
	}
}

func (o OpenAIClient) generate(
	ctx context.Context,
	prompt string,
	options ...GenerationOption,
) (*ImageGenerationResponse, error) {
	genOpts := GenerationOptions{
		Size:           o.options.model.DefaultSize,
		Quality:        o.options.model.DefaultQuality,
		ResponseFormat: "url",
		N:              1,
	}

	for _, opt := range options {
		opt(&genOpts)
	}

	params := openai.ImageGenerateParams{
		Prompt: prompt,
		Model:  openai.ImageModel(o.options.model.APIModel),
		N:      openai.Int(int64(genOpts.N)),
	}

	if genOpts.ResponseFormat != "" {
		params.ResponseFormat = openai.ImageGenerateParamsResponseFormat(genOpts.ResponseFormat)
	}

	if genOpts.Size != "" {
		params.Size = openai.ImageGenerateParamsSize(genOpts.Size)
	}

	if genOpts.Quality != "" {
		params.Quality = openai.ImageGenerateParamsQuality(genOpts.Quality)
	}

	if o.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *o.options.timeout)
		defer cancel()
	}

	response, err := o.client.Images.Generate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	results := make([]ImageGenerationResult, 0, len(response.Data))
	for _, img := range response.Data {
		result := ImageGenerationResult{
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

	return &ImageGenerationResponse{
		Images: results,
		Usage: ImageGenerationUsage{
			PromptTokens: 0,
		},
		Model: o.options.model.APIModel,
	}, nil
}
