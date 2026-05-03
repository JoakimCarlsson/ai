// Example image_generation_openai demonstrates generating images with OpenAI DALL·E.
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
		model.ProviderOpenAI,
		image.WithAPIKey(""),
		image.WithModel(
			model.OpenAIImageGenerationModels[model.DALLE3],
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.GenerateImage(
		context.Background(),
		"A serene mountain landscape at sunset with vibrant colors",
		image.WithSize("1024x1024"),
		image.WithQuality("hd"),
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
