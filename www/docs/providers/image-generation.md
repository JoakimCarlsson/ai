# Image Generation

The `image` modality. Vendors live under `image/`.

`image.GenerateImage(ctx, prompt)` takes only a prompt — every vendor knob
(size, aspect ratio, quality, response format, style, seed, safety, …) lives
on the vendor's `Options` and is set at construction. Image generation is
"configure once, prompt many" and vendor request bodies don't share enough
common shape to support a portable per-call surface.

## OpenAI

```go
import (
    "github.com/joakimcarlsson/ai/image"
    imageopenai "github.com/joakimcarlsson/ai/image/openai"
    "github.com/joakimcarlsson/ai/model"
)

client := imageopenai.NewGeneration(
    imageopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    imageopenai.WithModel(model.OpenAIImageGenerationModels[model.GPTImage15]),
    imageopenai.WithSize(imageopenai.Size1024x1024),
    imageopenai.WithQuality(imageopenai.QualityHigh),
    imageopenai.WithBackground(imageopenai.BackgroundTransparent),
    imageopenai.WithOutputFormat(imageopenai.OutputFormatPNG),
)

resp, err := client.GenerateImage(ctx, "A serene mountain landscape at sunset")
if err != nil {
    log.Fatal(err)
}

data, _ := image.DecodeBase64Image(resp.Images[0].ImageBase64)
os.WriteFile("output.png", data, 0644)
```

Full option set (typed enums — see the package's exported `Size`, `Quality`,
`Background`, `Moderation`, `OutputFormat` types):

```go
imageopenai.WithN(int)                                  // 1–10
imageopenai.WithSize(imageopenai.Size1024x1024)         // 1024x1024 | 1024x1536 | 1536x1024 | auto
imageopenai.WithQuality(imageopenai.QualityHigh)        // low | medium | high | auto
imageopenai.WithBackground(imageopenai.BackgroundAuto)  // transparent | opaque | auto — gpt-image-1.5 only (gpt-image-2 rejects)
imageopenai.WithModeration(imageopenai.ModerationAuto)  // auto | low
imageopenai.WithOutputFormat(imageopenai.OutputFormatPNG) // png | jpeg | webp
imageopenai.WithOutputCompression(int)                  // 0–100 — jpeg/webp only
imageopenai.WithUser(string)                            // end-user identifier
imageopenai.WithStreamingOptions(...)                   // partial-image count for streaming
```

Supported models: `gpt-image-1.5` and `gpt-image-2`. DALL-E 2/3 and gpt-image-1
(plus mini) are removed; pricing-registry entries dropped along with the
matching package code paths.

## Azure OpenAI

`image/azure` is to `image/openai` what `llm/azure` is to `llm/openai` — a thin
wrapper that reuses the OpenAI request building and overrides only endpoint and
auth. It resolves in three branches, mirroring `llm/azure`:

- no endpoint set → plain `image/openai` (optionally with `WithAPIKey`);
- an endpoint containing `/openai/v1` (the OpenAI-compatible surface) → routed
  through `image/openai` with `WithBaseURL`;
- a classic `https://<resource>.openai.azure.com` endpoint → the `api-key`
  header plus the `?api-version=` query param.

```go
import imageazure "github.com/joakimcarlsson/ai/image/azure"

client := imageazure.NewGeneration(
    imageazure.WithEndpoint("https://my-resource.openai.azure.com"),
    imageazure.WithAPIVersion("2025-04-01-preview"),
    imageazure.WithAPIKey(os.Getenv("AZURE_OPENAI_API_KEY")),
    imageazure.WithModel(model.OpenAIImageGenerationModels[model.GPTImage2]),
    imageazure.WithSize(imageazure.Size1024x1024),
    imageazure.WithOutputFormat(imageazure.OutputFormatPNG),
)

resp, err := client.GenerateImage(ctx, "A serene mountain landscape at sunset")
```

Auth resolution matches `llm/azure`: a static `api-key` is used when
`WithAPIKey` is set; otherwise the client falls back to
`DefaultAzureCredential` (Entra ID / managed identity) automatically — omit
`WithAPIKey` and ensure `az login` or a managed identity is available.

The `image/openai` enum types **and their values** are re-exported
(`imageazure.Size1024x1024`, `imageazure.QualityHigh`,
`imageazure.OutputFormatPNG`, …), and the full option set is forwarded:
`WithSize`, `WithQuality`, `WithN`, `WithBackground`, `WithModeration`,
`WithOutputFormat`, `WithOutputCompression`, `WithUser`, `WithExtraHeaders`,
`WithStreamingOptions`, `WithTimeout`. Returned clients are tracing-wrapped like
`image/openai`.

## Gemini / Imagen

```go
import imagegemini "github.com/joakimcarlsson/ai/image/gemini"

client := imagegemini.NewGeneration(
    imagegemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    imagegemini.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
    imagegemini.WithAspectRatio(imagegemini.AspectRatio16x9),
    imagegemini.WithN(2),
)

resp, err := client.GenerateImage(ctx, "A cyberpunk cityscape")
for i, img := range resp.Images {
    data, _ := image.DecodeBase64Image(img.ImageBase64)
    os.WriteFile(fmt.Sprintf("image_%d.png", i), data, 0644)
}
```

Full option set (Imagen-only fields are ignored when the active model is a
Gemini Image variant):

```go
import "google.golang.org/genai"

imagegemini.WithN(int32)                                          // Imagen: 1–4
imagegemini.WithAspectRatio(imagegemini.AspectRatio16x9)          // see imagegemini.AspectRatio*
imagegemini.WithNegativePrompt(string)                            // Imagen only
imagegemini.WithSeed(int32)                                       // Imagen only (requires AddWatermark=false)
imagegemini.WithPersonGeneration(genai.PersonGenerationAllowAdult) // both paths
imagegemini.WithSafetyFilterLevel(genai.SafetyFilterLevelBlockOnlyHigh) // Imagen only
imagegemini.WithLanguage(genai.ImagePromptLanguageEn)             // Imagen only
imagegemini.WithEnhancePrompt(bool)                               // Imagen only
imagegemini.WithImageSize(imagegemini.ImageSize2K)                // 1K | 2K | 4K — model-dependent
imagegemini.WithIncludeRAIReason(bool)                            // Imagen only
imagegemini.WithOutputMIMEType(imagegemini.OutputMIMETypePNG)     // image/png | image/jpeg — Imagen only
imagegemini.WithOutputCompressionQuality(int32)                   // 0–100 — Imagen jpeg only
```

## xAI Grok Imagine

```go
import imagexai "github.com/joakimcarlsson/ai/image/xai"

client := imagexai.NewGeneration(
    imagexai.WithAPIKey(os.Getenv("XAI_API_KEY")),
    imagexai.WithModel(model.XAIImageGenerationModels[model.XAIGrokImagineImage]),
    imagexai.WithAspectRatio(imagexai.AspectRatio16x9),
    imagexai.WithResolution(imagexai.Resolution2K),
    imagexai.WithResponseFormat(imagexai.ResponseFormatBase64),
)

resp, err := client.GenerateImage(ctx, "A neon-lit street market")
```

Full option set:

```go
imagexai.WithN(int)                                       // 1–10
imagexai.WithAspectRatio(imagexai.AspectRatio16x9)        // 14 values — see imagexai.AspectRatio*
imagexai.WithResolution(imagexai.Resolution2K)            // 1K | 2K
imagexai.WithResponseFormat(imagexai.ResponseFormatBase64) // url | b64_json
imagexai.WithUser(string)                                  // end-user identifier
```

## OpenRouter

Unlike the TTS and STT OpenRouter packages, `image/openrouter` is a full
implementation rather than a base-URL wrapper: OpenRouter's image endpoint is
`POST /api/v1/images`, a different path and body from OpenAI's
`/v1/images/generations`.

```go
import imageopenrouter "github.com/joakimcarlsson/ai/image/openrouter"

client := imageopenrouter.NewGeneration(
    imageopenrouter.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    imageopenrouter.WithModel(model.OpenRouterImageGenerationModels[model.OpenRouterSeedream45]),
    imageopenrouter.WithAspectRatio(imageopenrouter.AspectRatio16x9),
)

resp, err := client.GenerateImage(ctx, "A red panda astronaut")
data, _ := image.DecodeBase64Image(resp.Images[0].ImageBase64)
fmt.Println(resp.Images[0].MediaType) // image/png
fmt.Println(resp.Usage.Cost)         // 0.04 — dollars, as OpenRouter reported them
```

Full option set:

```go
imageopenrouter.WithN(int)                                          // 1–10, per-model ceiling
imageopenrouter.WithSize("2048x2048")                               // tier or explicit pixels
imageopenrouter.WithAspectRatio(imageopenrouter.AspectRatio16x9)    // 18 values
imageopenrouter.WithResolution(imageopenrouter.Resolution4K)        // 1K | 2K | 4K
imageopenrouter.WithQuality(imageopenrouter.QualityHigh)            // auto | low | medium | high
imageopenrouter.WithBackground(imageopenrouter.BackgroundTransparent) // auto | transparent | opaque
imageopenrouter.WithOutputFormat(imageopenrouter.OutputFormatWebP)  // png | jpeg | webp | svg
imageopenrouter.WithOutputCompression(int)                          // 0–100, webp/jpeg
imageopenrouter.WithSeed(int64)                                     // where supported
imageopenrouter.WithInputReferences(urls ...string)                 // image-to-image
imageopenrouter.WithProviderRouting(order []string, allowFallbacks bool)
imageopenrouter.WithRequestJSONField(key string, value any)         // escape hatch
imageopenrouter.WithHTTPClient(*http.Client)
imageopenrouter.WithExtraHeaders(map[string]string)                 // HTTP-Referer, X-Title
imageopenrouter.WithTimeout(time.Duration)
```

Not every model accepts every knob, and a model advertising a value does not mean
it accepts it in every combination. `seedream-4.5` lists `1K`, `2K` and `4K` but
enforces a floor of 3,686,400 output pixels, so `1K` and `2K` both draw an HTTP
400 at 16:9. Omitting `WithResolution` lets OpenRouter apply the model's own
default, which is why the examples above do. Query
`GET /api/v1/images/models` for a model's `supported_parameters` first.

There is no `WithModelFallbacks`. OpenRouter documents its `models` fallback
array for chat completions and `/api/v1/messages` only; on `/images` a
nonexistent primary answers `404 No model found` rather than falling through,
and the request schema silently drops fields it does not know. Fall back in
caller code with a second client instead. `WithProviderRouting` *is* documented
for this endpoint and is wired up.

### Using a model the registry does not define

`model.OpenRouterImageGenerationModels` carries 26 known-good defaults, but
OpenRouter routes more than that and adds new models weekly. You never have to
wait for a release to use one. `WithModelID` takes any raw OpenRouter id:

```go
client := imageopenrouter.NewGeneration(
    imageopenrouter.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    imageopenrouter.WithModelID("krea/krea-2-large"),
    imageopenrouter.WithAspectRatio(imageopenrouter.AspectRatio4x5),
)
```

That is shorthand for `WithModel(model.ImageGenerationModel{APIModel: id,
Provider: model.ProviderOpenRouter})`, so `Model()` reports the id and provider
and nothing else — no pricing, no supported-value lists. Generation works fine;
what breaks is anything that reads that metadata, such as a cost estimator or a
UI offering the caller a list of aspect ratios.

When you need those, describe the model yourself and it behaves exactly like a
registered entry, `DefaultAspectRatio` included:

```go
riverflowFast := model.ImageGenerationModel{
    ID:       "openrouter.riverflow-v2.5-fast",
    Name:     "OpenRouter – Riverflow V2.5 Fast",
    Provider: model.ProviderOpenRouter,
    APIModel: "sourceful/riverflow-v2.5-fast",
    Pricing: map[string]map[string]float64{
        "default": {"default": 0.019},
    },
    MaxPromptTokens:       4000,
    SupportedAspectRatios: []string{"1:1", "4:3", "16:9", "21:9", "auto"},
    DefaultAspectRatio:    "16:9",
    SupportedSizes:        []string{"1K", "2K"},
    DefaultSize:           "1K",
    SupportedQualities:    []string{"default"},
    DefaultQuality:        "default",
}

client := imageopenrouter.NewGeneration(
    imageopenrouter.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    imageopenrouter.WithModel(riverflowFast),
)
```

The real capability and pricing data for any id lives at
`GET /api/v1/images/models/<id>/endpoints`. `stt/openrouter` and
`tts/openrouter` expose the same `WithModelID` shorthand over
`model.TranscriptionModel` and `model.AudioModel`.

`examples/image/openrouter` runs all of this, including both custom-model forms.

`GenerateImageStreaming` is real here, not `ErrStreamingNotSupported`: models
whose descriptor reports `supports_streaming` (the `gpt-image-*` family today)
deliver `EventPartialImage` frames before the final `EventCompleted`.

Per-model capability data — including `SupportedAspectRatios` — lives on
`model.ImageGenerationModel`. Inspect it to know what a given model accepts:

```go
m := model.GeminiImageGenerationModels[model.Imagen4]
fmt.Println(m.SupportedAspectRatios) // [1:1 3:4 4:3 9:16 16:9]
```

## Streaming partial images (OpenAI gpt-image-*)

```go
client := imageopenai.NewGeneration(
    imageopenai.WithAPIKey(...),
    imageopenai.WithModel(model.OpenAIImageGenerationModels[model.GPTImage15]),
    imageopenai.WithStreamingOptions(imageopenai.StreamingOptions{PartialImages: 3}),
)

err := client.GenerateImageStreaming(ctx, prompt,
    func(event image.StreamEvent) error {
        switch event.Type {
        case image.EventPartialImage:
            data, _ := image.DecodeBase64Image(event.ImageBase64)
            os.WriteFile(fmt.Sprintf("partial_%d.png", event.PartialImageIndex), data, 0644)
        case image.EventCompleted:
            data, _ := image.DecodeBase64Image(event.ImageBase64)
            os.WriteFile("final.png", data, 0644)
        }
        return nil
    },
)
```

Returns `image.ErrStreamingNotSupported` if the model can't stream.

## Helpers

```go
// Download from URL
data, err := image.DownloadImage(resp.Images[0].ImageURL)

// Decode base64 payload
data, err := image.DecodeBase64Image(resp.Images[0].ImageBase64)
```
