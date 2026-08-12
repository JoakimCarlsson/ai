package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/image/gemini"
	imagegemini "github.com/joakimcarlsson/ai/image/gemini"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY is required")
	}

	client := imagegemini.NewGeneration(
		imagegemini.WithAPIKey(apiKey),
		imagegemini.WithModel(
			gemini.Models[gemini.Imagen4Fast],
		),
		imagegemini.WithAspectRatio(imagegemini.AspectRatio1x1),
	)

	resp, err := client.GenerateImage(
		context.Background(),
		"A clean flat illustration of Go modules as neatly stacked packages",
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

	const output = "gemini-image.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s with model %s\n", output, resp.Model)
}
