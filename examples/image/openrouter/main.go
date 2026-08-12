package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/image"
	"github.com/joakimcarlsson/ai/image/openrouter"
	imageopenrouter "github.com/joakimcarlsson/ai/image/openrouter"
)

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	generate(apiKey)
	custom(apiKey)
	customFullyDescribed(apiKey)
	stream(apiKey)
}

// generate renders one image and reports the two fields OpenRouter gives that
// the other image providers do not: the per-image media type and the
// dollar-denominated cost of the request.
//
// No WithResolution here on purpose. seedream-4.5 advertises 1K, 2K and 4K but
// enforces a floor of 3,686,400 output pixels, so both 1K and 2K are rejected at
// 16:9 with an HTTP 400. Omitting resolution lets OpenRouter apply the model's
// own default, which satisfies the floor.
func generate(apiKey string) {
	client := imageopenrouter.NewGeneration(
		imageopenrouter.WithAPIKey(apiKey),
		imageopenrouter.WithModel(
			openrouter.Models[openrouter.Seedream45],
		),
		imageopenrouter.WithAspectRatio(imageopenrouter.AspectRatio16x9),
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

// custom is the point of this example: using an OpenRouter image model that
// openrouter.Models does not catalogue.
//
// OpenRouter routes more image models than this repo tracks, and it adds new
// ones weekly. WithModelID takes any raw OpenRouter id, so a model that shipped
// after this package was last updated needs no code change here and no wait for
// a release. Find the id under GET /api/v1/images/models or on the model's page.
func custom(apiKey string) {
	client := imageopenrouter.NewGeneration(
		imageopenrouter.WithAPIKey(apiKey),
		imageopenrouter.WithModelID("krea/krea-2-large"),
		imageopenrouter.WithAspectRatio(imageopenrouter.AspectRatio4x5),
		imageopenrouter.WithResolution(imageopenrouter.Resolution1K),
	)

	resp, err := client.GenerateImage(
		context.Background(),
		"A brutalist concrete library at golden hour",
	)
	if err != nil {
		log.Fatal(err)
	}

	data, err := image.DecodeBase64Image(resp.Images[0].ImageBase64)
	if err != nil {
		log.Fatal(err)
	}

	const output = "openrouter-image-custom.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s (%s) via an uncatalogued model, cost $%.4f\n",
		output, resp.Images[0].MediaType, resp.Usage.Cost)
}

// customFullyDescribed is the same idea with the capability metadata filled in
// by hand.
//
// WithModelID leaves image.GenerationModel almost empty, which is fine for
// generating but means Model() reports no pricing and no supported-value lists.
// Anything that reads those — a cost estimator, a UI that offers the caller a
// list of aspect ratios — wants a fuller value. Build one with WithModel and it
// behaves exactly like a registered entry, including DefaultAspectRatio being
// applied when no explicit ratio is set.
//
// The values below are transcribed from
// GET /api/v1/images/models/sourceful/riverflow-v2.5-fast and its
// .../endpoints route, which is where any model's real capability and pricing
// data lives.
func customFullyDescribed(apiKey string) {
	riverflowFast := image.GenerationModel{
		ID:       "openrouter.riverflow-v2.5-fast",
		Name:     "OpenRouter – Riverflow V2.5 Fast",
		Provider: "openrouter",
		APIModel: "sourceful/riverflow-v2.5-fast",
		Pricing: map[string]map[string]float64{
			"default": {"default": 0.019},
		},
		MaxPromptTokens: 4000,
		SupportedAspectRatios: []string{
			"1:1", "4:3", "3:4", "3:2", "2:3", "16:9", "9:16", "21:9", "auto",
		},
		DefaultAspectRatio: "16:9",
		SupportedSizes:     []string{"1K", "2K"},
		DefaultSize:        "1K",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	}

	client := imageopenrouter.NewGeneration(
		imageopenrouter.WithAPIKey(apiKey),
		imageopenrouter.WithModel(riverflowFast),
	)

	m := client.Model()
	fmt.Printf("%s accepts %v, defaults to %s at $%.3f/image\n",
		m.Name, m.SupportedAspectRatios, m.DefaultAspectRatio,
		m.Pricing["default"]["default"])

	resp, err := client.GenerateImage(
		context.Background(),
		"A cable car crossing a foggy valley",
	)
	if err != nil {
		log.Fatal(err)
	}

	data, err := image.DecodeBase64Image(resp.Images[0].ImageBase64)
	if err != nil {
		log.Fatal(err)
	}

	const output = "openrouter-image-custom-described.png"
	if err := os.WriteFile(output, data, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s at the model's default %s\n",
		output, m.DefaultAspectRatio)
}

// stream shows the progressive-delivery path. Only models whose descriptor
// reports supports_streaming emit partial frames; gpt-image-2 does.
func stream(apiKey string) {
	client := imageopenrouter.NewGeneration(
		imageopenrouter.WithAPIKey(apiKey),
		imageopenrouter.WithModel(
			openrouter.Models[openrouter.GPTImage2],
		),
		imageopenrouter.WithQuality(imageopenrouter.QualityMedium),
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
