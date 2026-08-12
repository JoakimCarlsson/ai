package gemini

import (
	"github.com/joakimcarlsson/ai/image"
)

// Gemini provider plus Gemini and Imagen model IDs for this registry.
const (
	Gemini31FlashImagePreview string = "gemini-3.1-flash-image-preview"
	Gemini31FlashLiteImage    string = "gemini-3.1-flash-lite-image"
	Gemini3ProImage           string = "gemini-3-pro-image"
	Gemini25FlashImage        string = "gemini-2.5-flash-image"
	Imagen4                   string = "imagen-4.0"
	Imagen4Ultra              string = "imagen-4.0-ultra"
	Imagen4Fast               string = "imagen-4.0-fast"
)

// Models maps Gemini and Imagen image-generation model IDs to their configurations.
//
// Pricing source: https://ai.google.dev/gemini-api/docs/pricing.
// Fetched: 2026-07-26.
var Models = map[string]image.GenerationModel{
	Gemini25FlashImage: {
		ID:       Gemini25FlashImage,
		Name:     "Gemini 2.5 Flash Image",
		Provider: "gemini",
		APIModel: "gemini-2.5-flash-image",
		Pricing: map[string]map[string]float64{
			"16:9": {
				"default": 0.039,
			},
			"1:1": {
				"default": 0.039,
			},
			"3:4": {
				"default": 0.039,
			},
			"4:3": {
				"default": 0.039,
			},
			"9:16": {
				"default": 0.039,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"2:3",
			"3:2",
			"3:4",
			"4:3",
			"4:5",
			"5:4",
			"9:16",
			"16:9",
			"21:9",
		},
		DefaultAspectRatio: "1:1",
	},
	Gemini3ProImage: {
		ID:       Gemini3ProImage,
		Name:     "Gemini 3 Pro Image (Nano Banana Pro)",
		Provider: "gemini",
		APIModel: "gemini-3-pro-image-preview",
		Pricing: map[string]map[string]float64{
			"16:9": {
				"default": 0.134,
			},
			"1:1": {
				"default": 0.134,
			},
			"3:4": {
				"default": 0.134,
			},
			"4:3": {
				"default": 0.134,
			},
			"9:16": {
				"default": 0.134,
			},
		},
		MaxPromptTokens:    65536,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
		},
		DefaultAspectRatio: "1:1",
	},
	Gemini31FlashImagePreview: {
		ID:       Gemini31FlashImagePreview,
		Name:     "Gemini 3.1 Flash Image Preview (Nano Banana 2)",
		Provider: "gemini",
		APIModel: "gemini-3.1-flash-image",
		Pricing: map[string]map[string]float64{
			"16:9": {
				"default": 0.067,
			},
			"1:1": {
				"default": 0.067,
			},
			"1:4": {
				"default": 0.067,
			},
			"1:8": {
				"default": 0.067,
			},
			"3:4": {
				"default": 0.067,
			},
			"4:1": {
				"default": 0.067,
			},
			"4:3": {
				"default": 0.067,
			},
			"8:1": {
				"default": 0.067,
			},
			"9:16": {
				"default": 0.067,
			},
		},
		MaxPromptTokens:    131072,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
			"1:4",
			"4:1",
			"1:8",
			"8:1",
		},
		DefaultAspectRatio: "1:1",
	},
	Gemini31FlashLiteImage: {
		ID:       Gemini31FlashLiteImage,
		Name:     "Gemini 3.1 Flash Lite Image (Nano Banana 2 Lite)",
		Provider: "gemini",
		APIModel: "gemini-3.1-flash-lite-image",
		Pricing: map[string]map[string]float64{
			"16:9": {
				"default": 0.0336,
			},
			"1:1": {
				"default": 0.0336,
			},
			"1:4": {
				"default": 0.0336,
			},
			"1:8": {
				"default": 0.0336,
			},
			"3:4": {
				"default": 0.0336,
			},
			"4:1": {
				"default": 0.0336,
			},
			"4:3": {
				"default": 0.0336,
			},
			"8:1": {
				"default": 0.0336,
			},
			"9:16": {
				"default": 0.0336,
			},
		},
		MaxPromptTokens:    65536,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
			"1:4",
			"4:1",
			"1:8",
			"8:1",
		},
		DefaultAspectRatio: "1:1",
	},
	Imagen4: {
		ID:       Imagen4,
		Name:     "Imagen 4 [Deprecated]",
		Provider: "gemini",
		APIModel: "imagen-4.0-generate-001",
		Pricing: map[string]map[string]float64{
			"16:9": {
				"default": 0.04,
			},
			"1:1": {
				"default": 0.04,
			},
			"3:4": {
				"default": 0.04,
			},
			"4:3": {
				"default": 0.04,
			},
			"9:16": {
				"default": 0.04,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
		},
		DefaultAspectRatio: "1:1",
	},
	Imagen4Ultra: {
		ID:       Imagen4Ultra,
		Name:     "Imagen 4 Ultra [Deprecated]",
		Provider: "gemini",
		APIModel: "imagen-4.0-ultra-generate-001",
		Pricing: map[string]map[string]float64{
			"16:9": {
				"default": 0.06,
			},
			"1:1": {
				"default": 0.06,
			},
			"3:4": {
				"default": 0.06,
			},
			"4:3": {
				"default": 0.06,
			},
			"9:16": {
				"default": 0.06,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
		},
		DefaultAspectRatio: "1:1",
	},
	Imagen4Fast: {
		ID:       Imagen4Fast,
		Name:     "Imagen 4 Fast [Deprecated]",
		Provider: "gemini",
		APIModel: "imagen-4.0-fast-generate-001",
		Pricing: map[string]map[string]float64{
			"16:9": {
				"default": 0.02,
			},
			"1:1": {
				"default": 0.02,
			},
			"3:4": {
				"default": 0.02,
			},
			"4:3": {
				"default": 0.02,
			},
			"9:16": {
				"default": 0.02,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
		},
		DefaultAspectRatio: "1:1",
	},
}
