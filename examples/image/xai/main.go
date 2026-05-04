package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/image"
	imagexai "github.com/joakimcarlsson/ai/image/xai"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		log.Fatal("XAI_API_KEY is required")
	}

	client := imagexai.NewGeneration(
		imagexai.WithAPIKey(apiKey),
		imagexai.WithModel(
			model.XAIImageGenerationModels[model.XAIGrokImagineImage],
		),
		imagexai.WithAspectRatio(imagexai.AspectRatio16x9),
		imagexai.WithResolution(imagexai.Resolution2K),
		imagexai.WithResponseFormat(imagexai.ResponseFormatBase64),
	)

	resp, err := client.GenerateImage(
		context.Background(),
		"A neon-lit night market with steam rising from food stalls",
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

	const output = "xai-image.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s with model %s\n", output, resp.Model)
}
