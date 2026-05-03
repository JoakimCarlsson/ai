// Example image_generation demonstrates generating images with the image generation API.
package main

import (
	"context"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, err := image.NewImageGeneration(
		model.ProviderXAI,
		image.WithAPIKey(""),
		image.WithModel(
			model.XAIImageGenerationModels[model.XAIGrok2Image],
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.GenerateImage(
		context.Background(),
		"A serene mountain landscape at sunset with vibrant colors",
		image.WithResponseFormat("b64_json"),
	)
	if err != nil {
		log.Fatal(err)
	}

	imageData, err := image.DecodeBase64Image(
		response.Images[0].ImageBase64,
	)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("generated_image.png", imageData, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
