package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/image"
	imagegemini "github.com/joakimcarlsson/ai/image/gemini"
	imageopenai "github.com/joakimcarlsson/ai/image/openai"
	imagexai "github.com/joakimcarlsson/ai/image/xai"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, provider := newImageClient()

	resp, err := client.GenerateImage(
		context.Background(),
		"A simple diagram showing interchangeable AI providers",
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(resp.Images) == 0 || resp.Images[0].ImageBase64 == "" {
		log.Fatal("no image returned")
	}

	data, err := image.DecodeBase64Image(resp.Images[0].ImageBase64)
	if err != nil {
		log.Fatal(err)
	}

	output := provider + "-image.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[%s] saved %s with model %s\n", provider, output, resp.Model)
}

func newImageClient() (image.Generation, string) {
	switch provider := providerName(); provider {
	case "gemini":
		return imagegemini.NewGeneration(
			imagegemini.WithAPIKey(requiredEnv("GEMINI_API_KEY")),
			imagegemini.WithModel(
				model.GeminiImageGenerationModels[model.Imagen4Fast],
			),
			imagegemini.WithAspectRatio(imagegemini.AspectRatio1x1),
		), provider
	case "openai":
		return imageopenai.NewGeneration(
			imageopenai.WithAPIKey(requiredEnv("OPENAI_API_KEY")),
			imageopenai.WithModel(
				model.OpenAIImageGenerationModels[model.GPTImage15],
			),
			imageopenai.WithSize(imageopenai.Size1024x1024),
		), provider
	case "xai":
		return imagexai.NewGeneration(
			imagexai.WithAPIKey(requiredEnv("XAI_API_KEY")),
			imagexai.WithModel(
				model.XAIImageGenerationModels[model.XAIGrokImagineImage],
			),
			imagexai.WithAspectRatio(imagexai.AspectRatio1x1),
			imagexai.WithResponseFormat(imagexai.ResponseFormatBase64),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use openai, gemini, or xai)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "openai"
	}
	return provider
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
