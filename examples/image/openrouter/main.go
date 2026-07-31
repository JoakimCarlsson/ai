package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/image"
	imageopenrouter "github.com/joakimcarlsson/ai/image/openrouter"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	generate(apiKey)
	stream(apiKey)
}

// generate renders one image and reports the two fields OpenRouter gives that
// the other image providers do not: the per-image media type and the
// dollar-denominated cost of the request.
func generate(apiKey string) {
	client := imageopenrouter.NewGeneration(
		imageopenrouter.WithAPIKey(apiKey),
		imageopenrouter.WithModel(
			model.OpenRouterImageGenerationModels[model.OpenRouterSeedream45],
		),
		imageopenrouter.WithAspectRatio(imageopenrouter.AspectRatio16x9),
		imageopenrouter.WithResolution(imageopenrouter.Resolution2K),
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

	const output = "openrouter-image.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s (%s) with model %s, cost $%.4f\n",
		output, resp.Images[0].MediaType, resp.Model, resp.Usage.Cost)
}

// stream shows the progressive-delivery path. Only models whose descriptor
// reports supports_streaming emit partial frames; gpt-image-2 does.
func stream(apiKey string) {
	client := imageopenrouter.NewGeneration(
		imageopenrouter.WithAPIKey(apiKey),
		imageopenrouter.WithModel(
			model.OpenRouterImageGenerationModels[model.OpenRouterGPTImage2],
		),
		imageopenrouter.WithQuality(imageopenrouter.QualityMedium),
		imageopenrouter.WithModelFallbacks("bytedance-seed/seedream-4.5"),
	)

	var partials int
	var final string
	err := client.GenerateImageStreaming(
		context.Background(),
		"A lighthouse in a storm, painted in thick oils",
		func(event image.StreamEvent) error {
			switch event.Type {
			case image.EventPartialImage:
				partials++
				fmt.Printf("partial %d received\n", event.PartialImageIndex)
			case image.EventCompleted:
				final = event.ImageBase64
			}
			return nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	if final == "" {
		log.Fatal("stream ended without a completed image")
	}

	data, err := image.DecodeBase64Image(final)
	if err != nil {
		log.Fatal(err)
	}

	const output = "openrouter-image-streamed.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s after %d partial frames\n", output, partials)
}
