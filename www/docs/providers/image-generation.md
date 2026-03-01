# Image Generation

## OpenAI DALL-E 3

```go
import (
    "github.com/joakimcarlsson/ai/image_generation"
    "github.com/joakimcarlsson/ai/model"
)

client, err := image_generation.NewImageGeneration(
    model.ProviderOpenAI,
    image_generation.WithAPIKey("your-api-key"),
    image_generation.WithModel(model.OpenAIImageGenerationModels[model.DALLE3]),
)
if err != nil {
    log.Fatal(err)
}

response, err := client.GenerateImage(
    context.Background(),
    "A serene mountain landscape at sunset with vibrant colors",
    image_generation.WithSize("1024x1024"),
    image_generation.WithQuality("hd"),
    image_generation.WithResponseFormat("b64_json"),
)
if err != nil {
    log.Fatal(err)
}

imageData, _ := image_generation.DecodeBase64Image(response.Images[0].ImageBase64)
os.WriteFile("image.png", imageData, 0644)
```

## Google Gemini Imagen 4

```go
client, err := image_generation.NewImageGeneration(
    model.ProviderGemini,
    image_generation.WithAPIKey("your-api-key"),
    image_generation.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
)

response, err := client.GenerateImage(
    context.Background(),
    "A futuristic cityscape at night",
    image_generation.WithSize("16:9"),
    image_generation.WithN(4),
)
```

## xAI Grok 2 Image

```go
client, err := image_generation.NewImageGeneration(
    model.ProviderXAI,
    image_generation.WithAPIKey("your-api-key"),
    image_generation.WithModel(model.XAIImageGenerationModels[model.XAIGrok2Image]),
)

response, err := client.GenerateImage(
    context.Background(),
    "A robot playing chess",
    image_generation.WithResponseFormat("b64_json"),
)
```

## Client Options

```go
// OpenAI/xAI
client, err := image_generation.NewImageGeneration(
    model.ProviderOpenAI,
    image_generation.WithAPIKey("your-key"),
    image_generation.WithModel(model.OpenAIImageGenerationModels[model.DALLE3]),
    image_generation.WithTimeout(60*time.Second),
    image_generation.WithOpenAIOptions(
        image_generation.WithOpenAIBaseURL("custom-endpoint"),
    ),
)

// Gemini
client, err := image_generation.NewImageGeneration(
    model.ProviderGemini,
    image_generation.WithAPIKey("your-key"),
    image_generation.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
    image_generation.WithTimeout(60*time.Second),
    image_generation.WithGeminiOptions(
        image_generation.WithGeminiBackend(genai.BackendVertexAI),
    ),
)
```
