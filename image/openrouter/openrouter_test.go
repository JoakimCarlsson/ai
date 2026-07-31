package openrouter_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/image/openrouter"
	"github.com/joakimcarlsson/ai/model"
)

// pngB64 is a 1x1 PNG, base64-encoded the way OpenRouter returns image bytes.
const pngB64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR4nGP4" +
	"z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC"

// newServer records the decoded request body and path of the single request it
// expects, and replies with the supplied status and body.
func newServer(
	t *testing.T,
	body *map[string]any,
	path *string,
	status int,
	response string,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
			}
			if body != nil {
				if err := json.Unmarshal(raw, body); err != nil {
					t.Errorf("decode request body: %v", err)
				}
			}
			if path != nil {
				*path = r.URL.Path
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, response)
		}))
}

// TestGenerateImage covers the happy path: the documented response shape
// decodes into GenerationResponse with media_type and usage.cost preserved, the
// base64 payload survives image.DecodeBase64Image, and the request lands on
// /images rather than OpenAI's /images/generations.
func TestGenerateImage(t *testing.T) {
	var body map[string]any
	var path string
	srv := newServer(t, &body, &path, http.StatusOK, `{
		"created": 1748372400,
		"data": [{"b64_json": "`+pngB64+`", "media_type": "image/png"}],
		"usage": {"prompt_tokens": 16, "completion_tokens": 4175,
		          "total_tokens": 4191, "cost": 0.04}
	}`)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithAPIKey("test-key"),
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(
			model.OpenRouterImageGenerationModels[model.OpenRouterSeedream45],
		),
		openrouter.WithAspectRatio(openrouter.AspectRatio16x9),
		openrouter.WithResolution(openrouter.Resolution2K),
	)

	resp, err := client.GenerateImage(context.Background(), "a red panda")
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}

	if path != "/images" {
		t.Errorf("request path = %q, want /images", path)
	}
	if body["model"] != "bytedance-seed/seedream-4.5" {
		t.Errorf("model = %v, want bytedance-seed/seedream-4.5", body["model"])
	}
	if body["prompt"] != "a red panda" {
		t.Errorf("prompt = %v, want a red panda", body["prompt"])
	}
	if body["aspect_ratio"] != "16:9" {
		t.Errorf("aspect_ratio = %v, want 16:9", body["aspect_ratio"])
	}
	if body["resolution"] != "2K" {
		t.Errorf("resolution = %v, want 2K", body["resolution"])
	}
	if _, ok := body["stream"]; ok {
		t.Errorf("stream present on non-streaming request: %v", body["stream"])
	}

	if len(resp.Images) != 1 {
		t.Fatalf("len(Images) = %d, want 1", len(resp.Images))
	}
	if resp.Images[0].MediaType != "image/png" {
		t.Errorf("MediaType = %q, want image/png", resp.Images[0].MediaType)
	}
	if resp.Usage.Cost != 0.04 {
		t.Errorf("Usage.Cost = %v, want 0.04", resp.Usage.Cost)
	}
	if resp.Usage.PromptTokens != 16 {
		t.Errorf("Usage.PromptTokens = %d, want 16", resp.Usage.PromptTokens)
	}
	if resp.Model != "bytedance-seed/seedream-4.5" {
		t.Errorf("Model = %q, want bytedance-seed/seedream-4.5", resp.Model)
	}

	decoded, err := image.DecodeBase64Image(resp.Images[0].ImageBase64)
	if err != nil {
		t.Fatalf("DecodeBase64Image: %v", err)
	}
	if !strings.HasPrefix(string(decoded), "\x89PNG") {
		t.Errorf("decoded bytes are not a PNG: %q", decoded[:4])
	}
}

// TestGenerateImageArbitraryModel confirms a model id with no entry in the
// model package reaches the wire, matching llm/openrouter's documented
// behaviour.
func TestGenerateImageArbitraryModel(t *testing.T) {
	var body map[string]any
	srv := newServer(t, &body, nil, http.StatusOK,
		`{"data":[{"b64_json":"`+pngB64+`","media_type":"image/webp"}],
		  "usage":{"cost":0.01}}`)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(model.ImageGenerationModel{
			APIModel: "some-vendor/unreleased-image-model",
		}),
	)

	resp, err := client.GenerateImage(context.Background(), "anything")
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if body["model"] != "some-vendor/unreleased-image-model" {
		t.Errorf("model = %v, want some-vendor/unreleased-image-model",
			body["model"])
	}
	if _, ok := body["aspect_ratio"]; ok {
		t.Errorf("aspect_ratio sent for a model with no default: %v",
			body["aspect_ratio"])
	}
	if resp.Images[0].MediaType != "image/webp" {
		t.Errorf("MediaType = %q, want image/webp", resp.Images[0].MediaType)
	}
}

// TestWithModelID confirms the shorthand for uncatalogued models reaches the
// wire and tags the provider.
func TestWithModelID(t *testing.T) {
	var body map[string]any
	srv := newServer(t, &body, nil, http.StatusOK,
		`{"data":[{"b64_json":"`+pngB64+`"}],"usage":{"cost":0.03}}`)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModelID("black-forest-labs/flux.2-pro"),
	)

	if _, err := client.GenerateImage(context.Background(), "p"); err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if body["model"] != "black-forest-labs/flux.2-pro" {
		t.Errorf("model = %v, want black-forest-labs/flux.2-pro", body["model"])
	}
	if got := client.Model().Provider; got != model.ProviderOpenRouter {
		t.Errorf("Model().Provider = %q, want %q", got,
			model.ProviderOpenRouter)
	}
}

// TestWireOptions confirms every construction-time knob reaches the request
// body in OpenRouter's documented shape, including the routing helpers and the
// nested input_references objects.
func TestWireOptions(t *testing.T) {
	var body map[string]any
	srv := newServer(t, &body, nil, http.StatusOK,
		`{"data":[{"b64_json":"`+pngB64+`"}],"usage":{}}`)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(model.ImageGenerationModel{APIModel: "m"}),
		openrouter.WithN(3),
		openrouter.WithSize("2048x2048"),
		openrouter.WithAspectRatio(openrouter.AspectRatio1x1),
		openrouter.WithQuality(openrouter.QualityHigh),
		openrouter.WithBackground(openrouter.BackgroundTransparent),
		openrouter.WithOutputFormat(openrouter.OutputFormatWebP),
		openrouter.WithOutputCompression(80),
		openrouter.WithSeed(42),
		openrouter.WithInputReferences("https://example.com/photo.jpg"),
		openrouter.WithProviderRouting([]string{"openai", "azure"}, false),
		openrouter.WithModelFallbacks("a/b", "c/d"),
	)

	if _, err := client.GenerateImage(context.Background(), "p"); err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}

	for field, want := range map[string]any{
		"n":                  float64(3),
		"size":               "2048x2048",
		"aspect_ratio":       "1:1",
		"quality":            "high",
		"background":         "transparent",
		"output_format":      "webp",
		"output_compression": float64(80),
		"seed":               float64(42),
	} {
		if body[field] != want {
			t.Errorf("%s = %v (%T), want %v", field, body[field],
				body[field], want)
		}
	}

	refs, ok := body["input_references"].([]any)
	if !ok || len(refs) != 1 {
		t.Fatalf(
			"input_references = %v, want one entry",
			body["input_references"],
		)
	}
	ref, _ := refs[0].(map[string]any)
	if ref["type"] != "image_url" {
		t.Errorf("input_references[0].type = %v, want image_url", ref["type"])
	}
	url, _ := ref["image_url"].(map[string]any)
	if url["url"] != "https://example.com/photo.jpg" {
		t.Errorf("input_references[0].image_url.url = %v, want the photo URL",
			url["url"])
	}

	provider, ok := body["provider"].(map[string]any)
	if !ok {
		t.Fatalf("provider = %v (%T), want an object", body["provider"],
			body["provider"])
	}
	if provider["allow_fallbacks"] != false {
		t.Errorf("provider.allow_fallbacks = %v, want false",
			provider["allow_fallbacks"])
	}
	order, ok := provider["order"].([]any)
	if !ok || len(order) != 2 || order[0] != "openai" || order[1] != "azure" {
		t.Errorf("provider.order = %v, want [openai azure]", provider["order"])
	}

	models, ok := body["models"].([]any)
	if !ok || len(models) != 2 || models[0] != "a/b" || models[1] != "c/d" {
		t.Errorf("models = %v, want [a/b c/d]", body["models"])
	}
}

// TestGenerateImageError confirms both of OpenRouter's error envelopes surface
// their message rather than a bare status code.
func TestGenerateImageError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{
			name:   "openai shaped",
			status: http.StatusBadRequest,
			body: `{"error":{"message":"no endpoints found",` +
				`"code":"invalid_model"}}`,
			want: "no endpoints found",
		},
		{
			name:   "request validation",
			status: http.StatusBadRequest,
			body: `{"success":false,"error":{"name":"ZodError",` +
				`"message":"prompt: expected string"}}`,
			want: "prompt: expected string",
		},
		{
			name:   "not json",
			status: http.StatusBadGateway,
			body:   "upstream unavailable",
			want:   "upstream unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newServer(t, nil, nil, tt.status, tt.body)
			defer srv.Close()

			client := openrouter.NewGeneration(
				openrouter.WithBaseURL(srv.URL),
				openrouter.WithModel(model.ImageGenerationModel{APIModel: "m"}),
			)

			_, err := client.GenerateImage(context.Background(), "p")
			if err == nil {
				t.Fatal("GenerateImage succeeded, want an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to mention %q", err, tt.want)
			}
		})
	}
}

// newStreamServer replies with an SSE body assembled from the given data lines.
func newStreamServer(
	t *testing.T,
	body *map[string]any,
	events ...string,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
			}
			if body != nil {
				if err := json.Unmarshal(raw, body); err != nil {
					t.Errorf("decode request body: %v", err)
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			for _, event := range events {
				_, _ = io.WriteString(w, "data: "+event+"\n\n")
				if flusher != nil {
					flusher.Flush()
				}
			}
		}))
}

// TestGenerateImageStreaming confirms partial events arrive before the final
// one, that media_type rides along, and that stream:true is set on the request.
func TestGenerateImageStreaming(t *testing.T) {
	var body map[string]any
	srv := newStreamServer(t, &body,
		`{"type":"image_generation.partial_image","partial_image_index":0,`+
			`"b64_json":"`+pngB64+`"}`,
		`{"type":"image_generation.partial_image","partial_image_index":1,`+
			`"b64_json":"`+pngB64+`"}`,
		`{"type":"image_generation.completed","b64_json":"`+pngB64+`",`+
			`"media_type":"image/png","created":1748372400,`+
			`"usage":{"cost":0.011}}`,
		"[DONE]",
	)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(
			model.OpenRouterImageGenerationModels[model.OpenRouterGPTImage2],
		),
	)

	var events []image.StreamEvent
	err := client.GenerateImageStreaming(
		context.Background(),
		"a red panda",
		func(e image.StreamEvent) error {
			events = append(events, e)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("GenerateImageStreaming: %v", err)
	}

	if body["stream"] != true {
		t.Errorf("stream = %v, want true", body["stream"])
	}

	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	for i, want := range []image.StreamEventType{
		image.EventPartialImage,
		image.EventPartialImage,
		image.EventCompleted,
	} {
		if events[i].Type != want {
			t.Errorf("events[%d].Type = %q, want %q", i, events[i].Type, want)
		}
	}
	if events[1].PartialImageIndex != 1 {
		t.Errorf("events[1].PartialImageIndex = %d, want 1",
			events[1].PartialImageIndex)
	}
	if events[2].MediaType != "image/png" {
		t.Errorf(
			"events[2].MediaType = %q, want image/png",
			events[2].MediaType,
		)
	}
	if _, err := base64.StdEncoding.DecodeString(
		events[2].ImageBase64,
	); err != nil {
		t.Errorf("final event payload is not valid base64: %v", err)
	}
}

// TestGenerateImageStreamingLargeFrame guards the one thing a bufio.Scanner
// would break on: a single data: line larger than the scanner's 64 KiB token
// ceiling.
func TestGenerateImageStreamingLargeFrame(t *testing.T) {
	big := base64.StdEncoding.EncodeToString(
		[]byte(strings.Repeat("x", 256*1024)),
	)
	srv := newStreamServer(t, nil,
		`{"type":"image_generation.completed","b64_json":"`+big+`"}`,
		"[DONE]",
	)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(model.ImageGenerationModel{APIModel: "m"}),
	)

	var got string
	err := client.GenerateImageStreaming(
		context.Background(),
		"p",
		func(e image.StreamEvent) error {
			got = e.ImageBase64
			return nil
		},
	)
	if err != nil {
		t.Fatalf("GenerateImageStreaming: %v", err)
	}
	if got != big {
		t.Errorf("payload truncated: got %d bytes, want %d", len(got), len(big))
	}
}

// TestGenerateImageStreamingError confirms an error event aborts the stream
// with its message, and that a callback error propagates.
func TestGenerateImageStreamingError(t *testing.T) {
	srv := newStreamServer(t, nil,
		`{"type":"error","error":{"message":"Generation failed",`+
			`"code":"server_error"}}`,
		"[DONE]",
	)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(model.ImageGenerationModel{APIModel: "m"}),
	)

	err := client.GenerateImageStreaming(
		context.Background(),
		"p",
		func(image.StreamEvent) error { return nil },
	)
	if err == nil {
		t.Fatal("GenerateImageStreaming succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "Generation failed") {
		t.Errorf("error = %q, want it to mention the stream error", err)
	}
}

// TestGenerateImageStreamingCallbackError confirms a callback error stops the
// stream and is returned unchanged.
func TestGenerateImageStreamingCallbackError(t *testing.T) {
	srv := newStreamServer(t, nil,
		`{"type":"image_generation.partial_image","b64_json":"`+pngB64+`"}`,
		`{"type":"image_generation.completed","b64_json":"`+pngB64+`"}`,
		"[DONE]",
	)
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(model.ImageGenerationModel{APIModel: "m"}),
	)

	want := fmt.Errorf("caller gave up")
	calls := 0
	err := client.GenerateImageStreaming(
		context.Background(),
		"p",
		func(image.StreamEvent) error {
			calls++
			return want
		},
	)
	if err != want {
		t.Errorf("error = %v, want %v", err, want)
	}
	if calls != 1 {
		t.Errorf("callback called %d times, want 1", calls)
	}
}

// TestAuthHeaders confirms the API key and attribution headers are sent.
func TestAuthHeaders(t *testing.T) {
	var auth, title string
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			auth = r.Header.Get("Authorization")
			title = r.Header.Get("X-Title")
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":[],"usage":{}}`)
		}))
	defer srv.Close()

	client := openrouter.NewGeneration(
		openrouter.WithAPIKey("test-key"),
		openrouter.WithBaseURL(srv.URL),
		openrouter.WithModel(model.ImageGenerationModel{APIModel: "m"}),
		openrouter.WithExtraHeaders(map[string]string{"X-Title": "unit test"}),
	)

	if _, err := client.GenerateImage(context.Background(), "p"); err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if auth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want Bearer test-key", auth)
	}
	if title != "unit test" {
		t.Errorf("X-Title = %q, want unit test", title)
	}
}

// TestDefaultBaseURL pins the documented endpoint.
func TestDefaultBaseURL(t *testing.T) {
	if openrouter.DefaultBaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("DefaultBaseURL = %q, want https://openrouter.ai/api/v1",
			openrouter.DefaultBaseURL)
	}
}
