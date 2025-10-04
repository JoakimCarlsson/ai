package main

import (
	"context"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/image_generation"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, err := image_generation.NewImageGeneration(
		model.ProviderGemini,
		image_generation.WithAPIKey(""),
		image_generation.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.GenerateImage(
		context.Background(),
		"A serene mountain landscape at sunset with vibrant colors",
		image_generation.WithSize("16:9"),
	)
	if err != nil {
		log.Fatal(err)
	}

	imageData, err := image_generation.DecodeBase64Image(response.Images[0].ImageBase64)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("generated_image.png", imageData, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
