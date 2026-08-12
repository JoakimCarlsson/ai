package xai

import (
	"github.com/joakimcarlsson/ai/image"
)

// xAI provider identifier and Grok model IDs for this registry.
const (
	Grok2Image          string = "grok-2-image-1212"
	GrokImagineImage    string = "grok-imagine-image"
	GrokImagineImagePro string = "grok-imagine-image-pro"
)

// Models maps xAI image generation model IDs to their configurations.
//
// Pricing source: https://docs.x.ai/docs/models. Fetched: 2026-07-26.
// Grok Imagine pricing is flat per image regardless of resolution or quality.
var Models = map[string]image.GenerationModel{
	Grok2Image: {
		ID:       Grok2Image,
		Name:     "Grok 2 Image",
		Provider: "xai",
		APIModel: "grok-2-image-1212",
		Currency: "USD",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.07,
			},
		},
		MaxPromptTokens:    1000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
	},
	GrokImagineImage: {
		ID:       GrokImagineImage,
		Name:     "Grok Imagine Image",
		Provider: "xai",
		APIModel: "grok-imagine-image",
		Currency: "USD",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.02,
			},
		},
		MaxPromptTokens:    1000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"16:9",
			"9:16",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"2:1",
			"1:2",
			"19.5:9",
			"9:19.5",
			"20:9",
			"9:20",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	GrokImagineImagePro: {
		ID:       GrokImagineImagePro,
		Name:     "Grok Imagine Image Pro",
		Provider: "xai",
		APIModel: "grok-imagine-image-pro",
		Currency: "USD",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.07,
			},
		},
		MaxPromptTokens:    1000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"16:9",
			"9:16",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"2:1",
			"1:2",
			"19.5:9",
			"9:19.5",
			"20:9",
			"9:20",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
}
