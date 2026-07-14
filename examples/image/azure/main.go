package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/image"
	imageazure "github.com/joakimcarlsson/ai/image/azure"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	if endpoint == "" {
		log.Fatal("AZURE_OPENAI_ENDPOINT is required")
	}
	apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION")
	if apiVersion == "" {
		log.Fatal("AZURE_OPENAI_API_VERSION is required")
	}

	opts := []imageazure.Option{
		imageazure.WithEndpoint(endpoint),
		imageazure.WithAPIVersion(apiVersion),
		imageazure.WithModel(model.OpenAIImageGenerationModels[model.GPTImage2]),
		imageazure.WithSize(imageazure.Size1024x1024),
		imageazure.WithOutputFormat(imageazure.OutputFormatPNG),
	}

	if apiKey := os.Getenv("AZURE_OPENAI_API_KEY"); apiKey != "" {
		opts = append(opts, imageazure.WithAPIKey(apiKey))
	}

	client := imageazure.NewGeneration(opts...)

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

	const output = "azure-image.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s with model %s\n", output, resp.Model)
}
