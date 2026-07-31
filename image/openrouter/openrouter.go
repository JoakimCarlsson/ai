// Package openrouter provides an OpenRouter implementation of the
// [image.Generation] interface.
//
// Unlike [tts/openrouter] and [stt/openrouter], this is not a base-URL wrapper
// over the openai package. OpenRouter's image endpoint is POST /api/v1/images —
// a different path from OpenAI's /v1/images/generations, taking a flat
// {model, prompt, ...} body and answering per-image media types plus a
// dollar-denominated cost:
//
//	{"created":1748372400,
//	 "data":[{"b64_json":"...","media_type":"image/png"}],
//	 "usage":{"prompt_tokens":0,"completion_tokens":4175,"total_tokens":4175,
//	          "cost":0.04}}
//
// so the request and response are built here directly over net/http.
//
// OpenRouter exposes far more image models than the [model] package
// catalogues; pass any OpenRouter image model id via [WithModel] with a bare
// [model.ImageGenerationModel] even without a registered entry in [model]:
//
//	client := openrouter.NewGeneration(
//		openrouter.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
//		openrouter.WithModel(model.ImageGenerationModel{
//			APIModel: "bytedance-seed/seedream-4.5",
//		}),
//		openrouter.WithAspectRatio("16:9"),
//	)
//
// Not every model accepts every knob. Query
// GET /api/v1/images/models for a model's supported_parameters before setting
// resolution, quality, background or seed; OpenRouter rejects unsupported
// fields rather than ignoring them.
package openrouter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/model"
)

// DefaultBaseURL is the canonical OpenRouter API endpoint.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// AspectRatio is an OpenRouter aspect-ratio value. Stored as a typed string so
// a caller can pass a value outside the enum if OpenRouter ships one before
// this list is updated; the accepted set differs per model.
type AspectRatio string

// Aspect-ratio values seen across the routed image models.
const (
	AspectRatio1x1    AspectRatio = "1:1"
	AspectRatio1x2    AspectRatio = "1:2"
	AspectRatio2x1    AspectRatio = "2:1"
	AspectRatio2x3    AspectRatio = "2:3"
	AspectRatio3x2    AspectRatio = "3:2"
	AspectRatio3x4    AspectRatio = "3:4"
	AspectRatio4x3    AspectRatio = "4:3"
	AspectRatio4x5    AspectRatio = "4:5"
	AspectRatio5x4    AspectRatio = "5:4"
	AspectRatio9x16   AspectRatio = "9:16"
	AspectRatio16x9   AspectRatio = "16:9"
	AspectRatio9x19_5 AspectRatio = "9:19.5"
	AspectRatio19_5x9 AspectRatio = "19.5:9"
	AspectRatio9x20   AspectRatio = "9:20"
	AspectRatio20x9   AspectRatio = "20:9"
	AspectRatio9x21   AspectRatio = "9:21"
	AspectRatio21x9   AspectRatio = "21:9"
	AspectRatioAuto   AspectRatio = "auto"
)

// Resolution is an output-resolution tier.
type Resolution string

// Supported resolution tiers.
const (
	Resolution1K Resolution = "1K"
	Resolution2K Resolution = "2K"
	Resolution4K Resolution = "4K"
)

// Quality selects the rendering quality tier.
type Quality string

// Supported quality tiers.
const (
	QualityAuto   Quality = "auto"
	QualityLow    Quality = "low"
	QualityMedium Quality = "medium"
	QualityHigh   Quality = "high"
)

// Background controls background transparency.
type Background string

// Supported background values.
const (
	BackgroundAuto        Background = "auto"
	BackgroundTransparent Background = "transparent"
	BackgroundOpaque      Background = "opaque"
)

// OutputFormat selects the encoding of the returned bytes.
type OutputFormat string

// Supported output formats. Vector models answer svg.
const (
	OutputFormatPNG  OutputFormat = "png"
	OutputFormatJPEG OutputFormat = "jpeg"
	OutputFormatWebP OutputFormat = "webp"
	OutputFormatSVG  OutputFormat = "svg"
)

// Options configures the OpenRouter image generation client.
type Options struct {
	apiKey            string
	model             model.ImageGenerationModel
	baseURL           string
	httpClient        *http.Client
	timeout           *time.Duration
	extraHeaders      map[string]string
	n                 *int
	size              string
	aspectRatio       AspectRatio
	resolution        Resolution
	quality           Quality
	background        Background
	outputFormat      OutputFormat
	outputCompression *int
	seed              *int64
	inputReferences   []string
	extraBodyFields   map[string]any
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with OpenRouter.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the image generation model. Any OpenRouter model id works;
// a bare [model.ImageGenerationModel] with only APIModel set is enough.
func WithModel(m model.ImageGenerationModel) Option {
	return func(o *Options) { o.model = m }
}

// WithModelID selects a model by raw OpenRouter id, for the models
// [model.OpenRouterImageGenerationModels] does not catalogue. It is shorthand
// for [WithModel] with a bare [model.ImageGenerationModel], and it is the
// intended path for anything OpenRouter has routed since this package was last
// updated:
//
//	openrouter.WithModelID("black-forest-labs/flux.2-pro")
//
// Nothing is validated locally, so the per-model option ceilings recorded on a
// registered [model.ImageGenerationModel] are not available. Check the model's
// supported_parameters via GET /api/v1/images/models before setting resolution,
// quality, background or seed; OpenRouter rejects unsupported fields.
func WithModelID(id string) Option {
	return WithModel(model.ImageGenerationModel{
		APIModel: id,
		Provider: model.ProviderOpenRouter,
	})
}

// WithBaseURL overrides [DefaultBaseURL]. Useful for proxies or staging.
func WithBaseURL(baseURL string) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithHTTPClient supplies the HTTP client used for requests.
func WithHTTPClient(client *http.Client) Option {
	return func(o *Options) { o.httpClient = client }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithExtraHeaders adds custom HTTP headers to every request. OpenRouter reads
// HTTP-Referer and X-Title here for app attribution.
func WithExtraHeaders(headers map[string]string) Option {
	return func(o *Options) { o.extraHeaders = headers }
}

// WithN sets how many images to generate per request (1–10; the ceiling is
// per-model and not every model accepts n > 1).
func WithN(n int) Option {
	return func(o *Options) { o.n = &n }
}

// WithSize sets the size shorthand — either a tier or explicit pixels such as
// "2048x2048".
func WithSize(size string) Option {
	return func(o *Options) { o.size = size }
}

// WithAspectRatio sets the aspect ratio.
func WithAspectRatio(ratio AspectRatio) Option {
	return func(o *Options) { o.aspectRatio = ratio }
}

// WithResolution sets the output resolution tier.
func WithResolution(res Resolution) Option {
	return func(o *Options) { o.resolution = res }
}

// WithQuality sets the rendering quality tier.
func WithQuality(q Quality) Option {
	return func(o *Options) { o.quality = q }
}

// WithBackground controls background transparency.
func WithBackground(b Background) Option {
	return func(o *Options) { o.background = b }
}

// WithOutputFormat selects the encoding of the returned bytes.
func WithOutputFormat(format OutputFormat) Option {
	return func(o *Options) { o.outputFormat = format }
}

// WithOutputCompression sets the compression level (0–100) for webp and jpeg.
func WithOutputCompression(level int) Option {
	return func(o *Options) { o.outputCompression = &level }
}

// WithSeed sets the seed for deterministic generation, where the model
// supports it.
func WithSeed(seed int64) Option {
	return func(o *Options) { o.seed = &seed }
}

// WithInputReferences supplies reference images for image-to-image generation.
// Each reference is an HTTP(S) URL or a base64 data URL; the per-model ceiling
// is reported as input_references in the model descriptor.
func WithInputReferences(references ...string) Option {
	return func(o *Options) { o.inputReferences = references }
}

// WithRequestJSONField injects an arbitrary top-level field into the request
// body, for OpenRouter features newer than this package. Fields set here
// override the ones derived from the other options.
func WithRequestJSONField(key string, value any) Option {
	return func(o *Options) {
		if o.extraBodyFields == nil {
			o.extraBodyFields = make(map[string]any)
		}
		o.extraBodyFields[key] = value
	}
}

// WithProviderRouting sets OpenRouter's provider routing object, which the
// images endpoint documents alongside the generation fields. order lists
// provider slugs to try in preference order; allowFallbacks controls whether
// OpenRouter may fall back to providers outside that list when they are
// unavailable. See https://openrouter.ai/docs/features/provider-routing.
//
// There is deliberately no WithModelFallbacks counterpart here. OpenRouter
// documents the models fallback array for chat completions and /api/v1/messages
// only, and the images endpoint validates model before it routes: a nonexistent
// primary id answers 404 "No model found" rather than falling through to the
// list. The endpoint's request schema also ignores fields it does not know, so
// sending models would look like it worked while doing nothing. Fall back in
// caller code by constructing a second client instead.
func WithProviderRouting(order []string, allowFallbacks bool) Option {
	provider := map[string]any{"allow_fallbacks": allowFallbacks}
	if len(order) > 0 {
		provider["order"] = order
	}
	return WithRequestJSONField("provider", provider)
}

// Client implements [image.Generation] against OpenRouter's image API.
type Client struct {
	options    Options
	httpClient *http.Client
}

// NewGeneration constructs an OpenRouter image generation client. The returned
// [image.Generation] is wrapped with [image.WithTracing], so callers always get
// tracing spans and metrics.
func NewGeneration(opts ...Option) image.Generation {
	options := Options{baseURL: DefaultBaseURL}
	for _, o := range opts {
		o(&options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return image.WithTracing(&Client{
		options:    options,
		httpClient: httpClient,
	}, image.TracingAttrs{})
}

// Model returns the configured image generation model.
func (c *Client) Model() model.ImageGenerationModel {
	return c.options.model
}

type imageData struct {
	B64JSON       string `json:"b64_json"`
	URL           string `json:"url"`
	MediaType     string `json:"media_type"`
	RevisedPrompt string `json:"revised_prompt"`
}

type imageUsage struct {
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

type imageResponse struct {
	Created int64       `json:"created"`
	Model   string      `json:"model"`
	Data    []imageData `json:"data"`
	Usage   imageUsage  `json:"usage"`
}

// errorBody covers both error envelopes OpenRouter emits: the OpenAI-shaped
// {"error":{"message":...,"code":...}} and its own request-validation
// {"success":false,"error":{"name":...,"message":...}}.
type errorBody struct {
	Error struct {
		Name    string `json:"name"`
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (c *Client) requestBody(prompt string, stream bool) map[string]any {
	body := map[string]any{
		"model":  c.options.model.APIModel,
		"prompt": prompt,
	}
	if stream {
		body["stream"] = true
	}
	if c.options.n != nil {
		body["n"] = *c.options.n
	}
	if c.options.size != "" {
		body["size"] = c.options.size
	}

	aspectRatio := c.options.aspectRatio
	if aspectRatio == "" {
		aspectRatio = AspectRatio(c.options.model.DefaultAspectRatio)
	}
	if aspectRatio != "" {
		body["aspect_ratio"] = string(aspectRatio)
	}

	if c.options.resolution != "" {
		body["resolution"] = string(c.options.resolution)
	}
	if c.options.quality != "" {
		body["quality"] = string(c.options.quality)
	}
	if c.options.background != "" {
		body["background"] = string(c.options.background)
	}
	if c.options.outputFormat != "" {
		body["output_format"] = string(c.options.outputFormat)
	}
	if c.options.outputCompression != nil {
		body["output_compression"] = *c.options.outputCompression
	}
	if c.options.seed != nil {
		body["seed"] = *c.options.seed
	}
	if len(c.options.inputReferences) > 0 {
		refs := make([]map[string]any, 0, len(c.options.inputReferences))
		for _, ref := range c.options.inputReferences {
			refs = append(refs, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": ref},
			})
		}
		body["input_references"] = refs
	}

	maps.Copy(body, c.options.extraBodyFields)
	return body
}

func (c *Client) newRequest(
	ctx context.Context,
	body map[string]any,
	accept string,
) (*http.Request, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	endpoint := strings.TrimSuffix(c.options.baseURL, "/") + "/images"
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(raw),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", accept)
	if c.options.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.options.apiKey)
	}
	for k, v := range c.options.extraHeaders {
		req.Header.Set(k, v)
	}
	return req, nil
}

// responseError turns a non-2xx response into an error carrying OpenRouter's
// own message when it sent one.
func responseError(status int, raw []byte) error {
	var decoded errorBody
	if err := json.Unmarshal(raw, &decoded); err == nil &&
		decoded.Error.Message != "" {
		if decoded.Error.Code != nil {
			return fmt.Errorf(
				"openrouter image request failed (status %d, code %v): %s",
				status, decoded.Error.Code, decoded.Error.Message,
			)
		}
		return fmt.Errorf(
			"openrouter image request failed (status %d): %s",
			status, decoded.Error.Message,
		)
	}
	return fmt.Errorf(
		"openrouter image request failed (status %d): %s",
		status, strings.TrimSpace(string(raw)),
	)
}

func (c *Client) withTimeout(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	if c.options.timeout == nil {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, *c.options.timeout)
}

// GenerateImage performs a non-streaming image generation request.
func (c *Client) GenerateImage(
	ctx context.Context,
	prompt string,
) (*image.GenerationResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	req, err := c.newRequest(ctx, c.requestBody(prompt, false),
		"application/json")
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, responseError(resp.StatusCode, raw)
	}

	var decoded imageResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("failed to decode image response: %w", err)
	}

	results := make([]image.GenerationResult, 0, len(decoded.Data))
	for _, img := range decoded.Data {
		results = append(results, image.GenerationResult{
			ImageURL:      img.URL,
			ImageBase64:   img.B64JSON,
			MediaType:     img.MediaType,
			RevisedPrompt: img.RevisedPrompt,
		})
	}

	modelID := decoded.Model
	if modelID == "" {
		modelID = c.options.model.APIModel
	}

	return &image.GenerationResponse{
		Images: results,
		Usage: image.GenerationUsage{
			PromptTokens: decoded.Usage.PromptTokens,
			Cost:         decoded.Usage.Cost,
		},
		Model: modelID,
	}, nil
}

// streamEvent is one SSE payload from the image endpoint. The documented types
// are image_generation.partial_image, image_generation.completed and error.
type streamEvent struct {
	Type              string `json:"type"`
	B64JSON           string `json:"b64_json"`
	MediaType         string `json:"media_type"`
	PartialImageIndex int    `json:"partial_image_index"`
	Size              string `json:"size"`
	Quality           string `json:"quality"`
	Error             *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

// GenerateImageStreaming streams partial images during generation. OpenRouter
// delivers Server-Sent Events when stream is set; only some models support it
// (supports_streaming in the model descriptor), and the rest answer an error
// event, which is surfaced as an error rather than a silent single frame.
func (c *Client) GenerateImageStreaming(
	ctx context.Context,
	prompt string,
	callback image.StreamCallback,
) error {
	if callback == nil {
		return errors.New("openrouter: nil stream callback")
	}

	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	req, err := c.newRequest(ctx, c.requestBody(prompt, true),
		"text/event-stream")
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stream image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf(
				"openrouter image request failed (status %d)",
				resp.StatusCode,
			)
		}
		return responseError(resp.StatusCode, raw)
	}

	return c.consumeStream(resp.Body, callback)
}

// consumeStream reads the SSE body line by line. A bufio.Reader rather than a
// bufio.Scanner because a single data: line carries a whole base64 image and
// routinely exceeds the scanner's token ceiling.
func (c *Client) consumeStream(
	body io.Reader,
	callback image.StreamCallback,
) error {
	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadString('\n')
		if len(line) == 0 && err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("failed to read image stream: %w", err)
		}

		payload, ok := sseData(line)
		if !ok {
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("failed to read image stream: %w", err)
			}
			continue
		}
		if payload == "[DONE]" {
			return nil
		}

		var event streamEvent
		if uerr := json.Unmarshal([]byte(payload), &event); uerr != nil {
			return fmt.Errorf("failed to decode stream event: %w", uerr)
		}

		switch event.Type {
		case "image_generation.partial_image":
			err := callback(image.StreamEvent{
				Type:              image.EventPartialImage,
				ImageBase64:       event.B64JSON,
				PartialImageIndex: event.PartialImageIndex,
				Size:              event.Size,
				Quality:           event.Quality,
				MediaType:         event.MediaType,
			})
			if err != nil {
				return err
			}
		case "image_generation.completed":
			err := callback(image.StreamEvent{
				Type:        image.EventCompleted,
				ImageBase64: event.B64JSON,
				Size:        event.Size,
				Quality:     event.Quality,
				MediaType:   event.MediaType,
			})
			if err != nil {
				return err
			}
		case "error":
			if event.Error != nil {
				return fmt.Errorf(
					"openrouter image stream failed: %s",
					event.Error.Message,
				)
			}
			return errors.New("openrouter image stream failed")
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("failed to read image stream: %w", err)
		}
	}
}

// sseData extracts the payload of a `data:` line, reporting false for the
// comment, event-name and blank lines that frame an SSE stream.
func sseData(line string) (string, bool) {
	trimmed := strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(trimmed, "data:") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "data:")), true
}
