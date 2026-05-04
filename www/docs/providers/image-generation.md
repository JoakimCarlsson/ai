# Image Generation

The `image` modality. Vendors live under `image/`.

## OpenAI

```go
import (
    "github.com/joakimcarlsson/ai/image"
    imageopenai "github.com/joakimcarlsson/ai/image/openai"
    "github.com/joakimcarlsson/ai/model"
)

client := imageopenai.NewGeneration(
    imageopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    imageopenai.WithModel(model.OpenAIImageModels[model.DallE3]),
)

resp, err := client.GenerateImage(ctx, "A serene mountain landscape at sunset",
    image.WithSize("1024x1024"),
    image.WithQuality("hd"),
    image.WithResponseFormat("b64_json"),
)
if err != nil {
    log.Fatal(err)
}

data, _ := image.DecodeBase64Image(resp.Images[0].ImageBase64)
os.WriteFile("output.png", data, 0644)
```

## Gemini Imagen

```go
import imagegemini "github.com/joakimcarlsson/ai/image/gemini"

client := imagegemini.NewGeneration(
    imagegemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    imagegemini.WithModel(model.GeminiImageModels[model.Imagen4]),
)

resp, err := client.GenerateImage(ctx, "A cyberpunk cityscape",
    image.WithSize("16:9"),  // Gemini uses aspect ratios
    image.WithN(2),
)

for i, img := range resp.Images {
    data, _ := image.DecodeBase64Image(img.ImageBase64)
    os.WriteFile(fmt.Sprintf("image_%d.png", i), data, 0644)
}
```

## xAI Grok image

xAI's image generation uses the OpenAI-compatible API. Use `image/openai`:

```go
client := imageopenai.NewGeneration(
    imageopenai.WithAPIKey(os.Getenv("XAI_API_KEY")),
    imageopenai.WithBaseURL("https://api.x.ai/v1"),
    imageopenai.WithModel(model.XAIImageModels[model.XAIGrok2Image]),
)
```

## Streaming partial images (OpenAI gpt-image-1)

```go
err := client.GenerateImageStreaming(ctx, prompt,
    func(event image.ImageStreamEvent) error {
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

## Per-call options

```go
image.WithSize("1024x1024")        // OpenAI WIDTHxHEIGHT
image.WithSize("16:9")             // Gemini aspect ratio
image.WithQuality("hd")
image.WithResponseFormat("b64_json")  // or "url"
image.WithN(2)                     // images per call
```
