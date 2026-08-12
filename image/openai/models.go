package openai

import (
	"github.com/joakimcarlsson/ai/image"
)

// OpenAI provider plus chat, embedding, and image model IDs for this registry.
const (
	GPTImage15 string = "gpt-image-1.5"
	GPTImage2  string = "gpt-image-2"
)

// Models maps OpenAI image generation model IDs to their configurations.
//
// Pricing source: https://developers.openai.com/api/docs/pricing (per-image
// quality tiers not published on the page; carried forward).
// Fetched: not re-verified in the 2026-07-26 sweep.
var Models = map[string]image.GenerationModel{
	GPTImage15: {
		ID:       GPTImage15,
		Name:     "GPT Image 1.5",
		Provider: "openai",
		APIModel: "gpt-image-1.5",
		Currency: "USD",
		Pricing: map[string]map[string]float64{
			"1024x1024": {
				"high":   0.133,
				"low":    0.009,
				"medium": 0.034,
			},
			"1024x1536": {
				"high":   0.2,
				"low":    0.013,
				"medium": 0.05,
			},
			"1536x1024": {
				"high":   0.2,
				"low":    0.013,
				"medium": 0.05,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1024x1024", "1024x1536", "1536x1024"},
		DefaultSize:        "1024x1024",
		SupportedQualities: []string{"low", "medium", "high"},
		DefaultQuality:     "medium",
		SupportsStreaming:  true,
	},
	GPTImage2: {
		ID:       GPTImage2,
		Name:     "GPT Image 2",
		Provider: "openai",
		APIModel: "gpt-image-2",
		Currency: "USD",
		Pricing: map[string]map[string]float64{
			"1024x1024": {
				"high":   0.211,
				"low":    0.006,
				"medium": 0.053,
			},
			"1024x1536": {
				"high":   0.165,
				"low":    0.005,
				"medium": 0.041,
			},
			"1536x1024": {
				"high":   0.165,
				"low":    0.005,
				"medium": 0.041,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1024x1024", "1024x1536", "1536x1024"},
		DefaultSize:        "1024x1024",
		SupportedQualities: []string{"low", "medium", "high"},
		DefaultQuality:     "medium",
		SupportsStreaming:  true,
	},
}
