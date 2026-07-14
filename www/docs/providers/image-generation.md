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
