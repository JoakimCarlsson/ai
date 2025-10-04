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

type XAIClient struct {
	client  openai.Client
	options imageGenerationClientOptions
}

type xaiOptions struct {
	baseURL      string
	extraHeaders map[string]string
}

// XAIOption is a function that configures xAI-specific options.
type XAIOption func(*xaiOptions)

// WithXAIBaseURL sets a custom base URL for the xAI API endpoint.
// The default is "https://api.x.ai/v1".
func WithXAIBaseURL(baseURL string) XAIOption {
	return func(options *xaiOptions) {
		options.baseURL = baseURL
	}
}

// WithXAIExtraHeaders adds custom HTTP headers to xAI API requests.
func WithXAIExtraHeaders(headers map[string]string) XAIOption {
	return func(options *xaiOptions) {
		options.extraHeaders = headers
	}
}

func newXAIClient(opts imageGenerationClientOptions) XAIClient {
	xaiOpts := xaiOptions{
		baseURL:      "https://api.x.ai/v1",
		extraHeaders: make(map[string]string),
	}

	for _, o := range opts.xaiOptions {
		o(&xaiOpts)
	}

	clientOpts := []option.RequestOption{
		option.WithAPIKey(opts.apiKey),
		option.WithBaseURL(xaiOpts.baseURL),
	}

	for k, v := range xaiOpts.extraHeaders {
		clientOpts = append(clientOpts, option.WithHeader(k, v))
	}

	client := openai.NewClient(clientOpts...)

	return XAIClient{
		client:  client,
		options: opts,
	}
}

func (x XAIClient) generate(
	ctx context.Context,
	prompt string,
	options ...GenerationOption,
) (*ImageGenerationResponse, error) {
	genOpts := GenerationOptions{
		Size:           x.options.model.DefaultSize,
		ResponseFormat: "url",
		N:              1,
	}

	for _, o := range options {
		o(&genOpts)
	}

	params := openai.ImageGenerateParams{
		Prompt: prompt,
		Model:  openai.ImageModel(x.options.model.APIModel),
		N:      openai.Int(int64(genOpts.N)),
	}

	if genOpts.ResponseFormat != "" {
		params.ResponseFormat = openai.ImageGenerateParamsResponseFormat(genOpts.ResponseFormat)
	}

	if x.options.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *x.options.timeout)
		defer cancel()
	}

	response, err := x.client.Images.Generate(ctx, params)
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
		Model: x.options.model.APIModel,
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
		return nil, fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
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
