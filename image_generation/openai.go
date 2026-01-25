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
	client     openai.Client
	options    imageGenerationClientOptions
	openaiOpts openaiOptions
}

type openaiOptions struct {
	baseURL          string
	extraHeaders     map[string]string
	streamingOptions OpenAIStreamingOptions
}

// OpenAIStreamingOptions contains OpenAI-specific options for streaming image generation.
type OpenAIStreamingOptions struct {
	// PartialImages specifies the number of partial images to receive during streaming (0-3).
	// If set to 0, only the final image will be received.
	// You may receive fewer partial images than requested if the full image is generated quickly.
	PartialImages int
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

// WithOpenAIStreamingOptions sets OpenAI-specific streaming configuration.
// Use this to configure the number of partial images to receive during streaming.
func WithOpenAIStreamingOptions(opts OpenAIStreamingOptions) OpenAIOption {
	return func(options *openaiOptions) {
		options.streamingOptions = opts
	}
}

func newOpenAIClient(opts imageGenerationClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{
		baseURL:      "",
		extraHeaders: make(map[string]string),
		streamingOptions: OpenAIStreamingOptions{
			PartialImages: 2,
		},
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
		client:     client,
		options:    opts,
		openaiOpts: openaiOpts,
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

	if genOpts.ResponseFormat != "" && o.options.model.APIModel != "gpt-image-1" &&
		o.options.model.APIModel != "gpt-image-1.5" && o.options.model.APIModel != "gpt-image-1-mini" {
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

func (o OpenAIClient) generateStreaming(
	ctx context.Context,
	prompt string,
	callback StreamCallback,
	options ...GenerationOption,
) error {
	if !o.options.model.SupportsStreaming {
		return ErrStreamingNotSupported
	}

	genOpts := GenerationOptions{
		Size:    o.options.model.DefaultSize,
		Quality: o.options.model.DefaultQuality,
		N:       1,
	}

	for _, opt := range options {
		opt(&genOpts)
	}

	params := openai.ImageGenerateParams{
		Prompt:        prompt,
		Model:         openai.ImageModel(o.options.model.APIModel),
		N:             openai.Int(int64(genOpts.N)),
		PartialImages: openai.Int(int64(o.openaiOpts.streamingOptions.PartialImages)),
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

	stream := o.client.Images.GenerateStreaming(ctx, params)

	for stream.Next() {
		event := stream.Current()

		switch event.Type {
		case "image_generation.partial_image":
			if err := callback(ImageStreamEvent{
				Type:              EventPartialImage,
				ImageBase64:       event.B64JSON,
				PartialImageIndex: int(event.PartialImageIndex),
				Size:              event.Size,
				Quality:           event.Quality,
			}); err != nil {
				return fmt.Errorf("callback error on partial image: %w", err)
			}

		case "image_generation.completed":
			if err := callback(ImageStreamEvent{
				Type:        EventCompleted,
				ImageBase64: event.B64JSON,
				Size:        event.Size,
				Quality:     event.Quality,
			}); err != nil {
				return fmt.Errorf("callback error on completed image: %w", err)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("streaming error: %w", err)
	}

	return nil
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
