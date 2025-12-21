package image_generation

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

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

	if genOpts.ResponseFormat != "" && o.options.model.APIModel != "gpt-image-1" && o.options.model.APIModel != "gpt-image-1.5" {
		params.ResponseFormat = openai.ImageGenerateParamsResponseFormat(
			genOpts.ResponseFormat,
		)
	}

	if genOpts.Size != "" && len(o.options.model.SupportedSizes) > 0 {
		params.Size = openai.ImageGenerateParamsSize(genOpts.Size)
	}

	if genOpts.Quality != "" && genOpts.Quality != "default" &&
		len(o.options.model.SupportedQualities) > 1 {
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
