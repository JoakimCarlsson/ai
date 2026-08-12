package openrouter

import (
	"github.com/joakimcarlsson/ai/image"
)

// OpenRouter image generation model IDs.
const (
	GPTImage2               string = "openrouter.gpt-image-2"
	GPTImage1               string = "openrouter.gpt-image-1"
	GPTImage1Mini           string = "openrouter.gpt-image-1-mini"
	GPT5Image               string = "openrouter.gpt-5-image"
	GPT5ImageMini           string = "openrouter.gpt-5-image-mini"
	GPT54Image2             string = "openrouter.gpt-5.4-image-2"
	Gemini25FlashImage      string = "openrouter.gemini-2.5-flash-image"
	Gemini3ProImage         string = "openrouter.gemini-3-pro-image"
	Gemini31FlashImage      string = "openrouter.gemini-3.1-flash-image"
	Gemini31FlashLiteImage  string = "openrouter.gemini-3.1-flash-lite-image"
	Seedream45              string = "openrouter.seedream-4.5"
	Flux2Pro                string = "openrouter.flux.2-pro"
	Flux2Max                string = "openrouter.flux.2-max"
	Flux2Flex               string = "openrouter.flux.2-flex"
	Flux2Klein4B            string = "openrouter.flux.2-klein-4b"
	GrokImagineImageQuality string = "openrouter.grok-imagine-image-quality"
	MAIImage25              string = "openrouter.mai-image-2.5"
	MAIImage25Pro           string = "openrouter.mai-image-2.5-pro"
	RiverflowV25Pro         string = "openrouter.riverflow-v2.5-pro"
	RiverflowV25Fast        string = "openrouter.riverflow-v2.5-fast"
	RecraftV41              string = "openrouter.recraft-v4.1"
	RecraftV41Pro           string = "openrouter.recraft-v4.1-pro"
	RecraftV41Vector        string = "openrouter.recraft-v4.1-vector"
	RecraftV41ProVector     string = "openrouter.recraft-v4.1-pro-vector"
	RecraftV4               string = "openrouter.recraft-v4"
	RecraftV4Vector         string = "openrouter.recraft-v4-vector"
)

// Models maps OpenRouter image model IDs to their
// configurations.
//
// These are known-good defaults, not a mirror of OpenRouter's catalogue:
// OpenRouter routes more image models than this package catalogues and the list
// moves weekly. Any OpenRouter image model id works with a bare
// [GenerationModel] even without an entry here — see the image/openrouter
// package docs.
//
// Capability source: https://openrouter.ai/api/v1/images/models. Pricing
// source: the per-model .../endpoints route. Fetched: 2026-07-31.
//
// Pricing is only populated where OpenRouter publishes a flat per-image rate,
// or where the equivalent upstream model has a published per-image estimate. It is left nil for the models OpenRouter bills
// per output token or per megapixel, and for the models it bills at several
// per-image tiers without saying which tier a given request lands in — a made-up
// per-image figure would be worse than none. Read usage.Cost off the response
// for what a request actually cost.
//
// DefaultSize is deliberately left empty across these entries. SupportedSizes
// records the resolution enum a model advertises, but advertising a tier is not
// the same as accepting it: seedream-4.5 lists 1K, 2K and 4K yet rejects both 1K
// and 2K at 16:9 with "requires at least 3,686,400 output pixels". Omitting
// resolution lets OpenRouter apply the model's real default, which is the only
// value verified to work for every entry here.
var Models = map[string]image.GenerationModel{
	GPTImage2: {
		ID:       GPTImage2,
		Name:     "OpenRouter – GPT Image 2",
		Provider: "openrouter",
		APIModel: "openai/gpt-image-2",
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
		MaxPromptTokens: 4000,
		SupportedQualities: []string{
			"auto",
			"low",
			"medium",
			"high",
		},
		DefaultQuality: "auto",
		SupportedAspectRatios: []string{
			"1:1",
			"3:2",
			"2:3",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
		SupportsStreaming:  true,
	},
	GPTImage1: {
		ID:              GPTImage1,
		Name:            "OpenRouter – GPT Image 1",
		Provider:        "openrouter",
		APIModel:        "openai/gpt-image-1",
		MaxPromptTokens: 4000,
		SupportedQualities: []string{
			"auto",
			"low",
			"medium",
			"high",
		},
		DefaultQuality: "auto",
		SupportedAspectRatios: []string{
			"1:1",
			"3:2",
			"2:3",
			"auto",
		},
		DefaultAspectRatio: "1:1",
		SupportsStreaming:  true,
	},
	GPTImage1Mini: {
		ID:              GPTImage1Mini,
		Name:            "OpenRouter – GPT Image 1 mini",
		Provider:        "openrouter",
		APIModel:        "openai/gpt-image-1-mini",
		MaxPromptTokens: 4000,
		SupportedQualities: []string{
			"auto",
			"low",
			"medium",
			"high",
		},
		DefaultQuality: "auto",
		SupportedAspectRatios: []string{
			"1:1",
			"3:2",
			"2:3",
			"auto",
		},
		DefaultAspectRatio: "1:1",
		SupportsStreaming:  true,
	},
	GPT5Image: {
		ID:              GPT5Image,
		Name:            "OpenRouter – GPT-5 Image",
		Provider:        "openrouter",
		APIModel:        "openai/gpt-5-image",
		MaxPromptTokens: 4000,
		SupportedQualities: []string{
			"auto",
			"low",
			"medium",
			"high",
		},
		DefaultQuality:    "auto",
		SupportsStreaming: true,
	},
	GPT5ImageMini: {
		ID:              GPT5ImageMini,
		Name:            "OpenRouter – GPT-5 Image mini",
		Provider:        "openrouter",
		APIModel:        "openai/gpt-5-image-mini",
		MaxPromptTokens: 4000,
		SupportedQualities: []string{
			"auto",
			"low",
			"medium",
			"high",
		},
		DefaultQuality:    "auto",
		SupportsStreaming: true,
	},
	GPT54Image2: {
		ID:              GPT54Image2,
		Name:            "OpenRouter – GPT-5.4 Image 2",
		Provider:        "openrouter",
		APIModel:        "openai/gpt-5.4-image-2",
		MaxPromptTokens: 4000,
		SupportedQualities: []string{
			"auto",
			"low",
			"medium",
			"high",
		},
		DefaultQuality:    "auto",
		SupportsStreaming: true,
	},
	Gemini25FlashImage: {
		ID:       Gemini25FlashImage,
		Name:     "OpenRouter – Gemini 2.5 Flash Image (Nano Banana)",
		Provider: "openrouter",
		APIModel: "google/gemini-2.5-flash-image",
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
		Name:     "OpenRouter – Gemini 3 Pro Image (Nano Banana Pro)",
		Provider: "openrouter",
		APIModel: "google/gemini-3-pro-image",
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
		SupportedSizes:     []string{"1K", "2K", "4K"},
		DefaultSize:        "1K",
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
	Gemini31FlashImage: {
		ID:       Gemini31FlashImage,
		Name:     "OpenRouter – Gemini 3.1 Flash Image (Nano Banana 2)",
		Provider: "openrouter",
		APIModel: "google/gemini-3.1-flash-image",
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
		MaxPromptTokens: 131072,
		SupportedSizes: []string{
			"512",
			"1K",
			"2K",
			"4K",
		},
		DefaultSize:        "1K",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"1:4",
			"1:8",
			"2:3",
			"3:2",
			"3:4",
			"4:1",
			"4:3",
			"4:5",
			"5:4",
			"8:1",
			"9:16",
			"16:9",
			"21:9",
		},
		DefaultAspectRatio: "1:1",
	},
	Gemini31FlashLiteImage: {
		ID:       Gemini31FlashLiteImage,
		Name:     "OpenRouter – Gemini 3.1 Flash Lite Image (Nano Banana 2 Lite)",
		Provider: "openrouter",
		APIModel: "google/gemini-3.1-flash-lite-image",
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
		SupportedSizes:     []string{"1K"},
		DefaultSize:        "1K",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"1:4",
			"1:8",
			"2:3",
			"3:2",
			"3:4",
			"4:1",
			"4:3",
			"4:5",
			"5:4",
			"8:1",
			"9:16",
			"16:9",
			"21:9",
		},
		DefaultAspectRatio: "1:1",
	},
	Seedream45: {
		ID:       Seedream45,
		Name:     "OpenRouter – Seedream 4.5",
		Provider: "openrouter",
		APIModel: "bytedance-seed/seedream-4.5",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.04,
			},
		},
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1K", "2K", "4K"},
		DefaultSize:        "1K",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"1:2",
			"2:1",
			"2:3",
			"3:2",
			"3:4",
			"4:3",
			"4:5",
			"5:4",
			"9:16",
			"16:9",
			"9:19.5",
			"19.5:9",
			"9:20",
			"20:9",
			"9:21",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	Flux2Pro: {
		ID:                 Flux2Pro,
		Name:               "OpenRouter – FLUX.2 Pro",
		Provider:           "openrouter",
		APIModel:           "black-forest-labs/flux.2-pro",
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"16:9",
			"9:16",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	Flux2Max: {
		ID:                 Flux2Max,
		Name:               "OpenRouter – FLUX.2 Max",
		Provider:           "openrouter",
		APIModel:           "black-forest-labs/flux.2-max",
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"16:9",
			"9:16",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	Flux2Flex: {
		ID:                 Flux2Flex,
		Name:               "OpenRouter – FLUX.2 Flex",
		Provider:           "openrouter",
		APIModel:           "black-forest-labs/flux.2-flex",
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"16:9",
			"9:16",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	Flux2Klein4B: {
		ID:                 Flux2Klein4B,
		Name:               "OpenRouter – FLUX.2 Klein 4B",
		Provider:           "openrouter",
		APIModel:           "black-forest-labs/flux.2-klein-4b",
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"16:9",
			"9:16",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	GrokImagineImageQuality: {
		ID:                 GrokImagineImageQuality,
		Name:               "OpenRouter – Grok Imagine Image Quality",
		Provider:           "openrouter",
		APIModel:           "x-ai/grok-imagine-image-quality",
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1K", "2K"},
		DefaultSize:        "1K",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"3:4",
			"4:3",
			"9:16",
			"16:9",
			"2:3",
			"3:2",
			"9:19.5",
			"19.5:9",
			"9:20",
			"20:9",
			"1:2",
			"2:1",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	MAIImage25: {
		ID:                 MAIImage25,
		Name:               "OpenRouter – MAI-Image-2.5",
		Provider:           "openrouter",
		APIModel:           "microsoft/mai-image-2.5",
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"3:2",
			"2:3",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	MAIImage25Pro: {
		ID:                 MAIImage25Pro,
		Name:               "OpenRouter – MAI-Image-2.5 Pro",
		Provider:           "openrouter",
		APIModel:           "microsoft/mai-image-2.5-pro",
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"3:2",
			"2:3",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RiverflowV25Pro: {
		ID:                 RiverflowV25Pro,
		Name:               "OpenRouter – Riverflow V2.5 Pro",
		Provider:           "openrouter",
		APIModel:           "sourceful/riverflow-v2.5-pro",
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1K", "2K", "4K"},
		DefaultSize:        "1K",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"16:9",
			"9:16",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RiverflowV25Fast: {
		ID:                 RiverflowV25Fast,
		Name:               "OpenRouter – Riverflow V2.5 Fast",
		Provider:           "openrouter",
		APIModel:           "sourceful/riverflow-v2.5-fast",
		MaxPromptTokens:    4000,
		SupportedSizes:     []string{"1K", "2K"},
		DefaultSize:        "1K",
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"3:2",
			"2:3",
			"16:9",
			"9:16",
			"21:9",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RecraftV41: {
		ID:       RecraftV41,
		Name:     "OpenRouter – Recraft V4.1",
		Provider: "openrouter",
		APIModel: "recraft/recraft-v4.1",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.035,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RecraftV41Pro: {
		ID:       RecraftV41Pro,
		Name:     "OpenRouter – Recraft V4.1 Pro",
		Provider: "openrouter",
		APIModel: "recraft/recraft-v4.1-pro",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.21,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RecraftV41Vector: {
		ID:       RecraftV41Vector,
		Name:     "OpenRouter – Recraft V4.1 Vector",
		Provider: "openrouter",
		APIModel: "recraft/recraft-v4.1-vector",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.08,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RecraftV41ProVector: {
		ID:       RecraftV41ProVector,
		Name:     "OpenRouter – Recraft V4.1 Pro Vector",
		Provider: "openrouter",
		APIModel: "recraft/recraft-v4.1-pro-vector",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.3,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RecraftV4: {
		ID:       RecraftV4,
		Name:     "OpenRouter – Recraft V4",
		Provider: "openrouter",
		APIModel: "recraft/recraft-v4",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.04,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
	RecraftV4Vector: {
		ID:       RecraftV4Vector,
		Name:     "OpenRouter – Recraft V4 Vector",
		Provider: "openrouter",
		APIModel: "recraft/recraft-v4-vector",
		Pricing: map[string]map[string]float64{
			"default": {
				"default": 0.08,
			},
		},
		MaxPromptTokens:    4000,
		SupportedQualities: []string{"default"},
		DefaultQuality:     "default",
		SupportedAspectRatios: []string{
			"1:1",
			"4:3",
			"3:4",
			"16:9",
			"9:16",
			"auto",
		},
		DefaultAspectRatio: "1:1",
	},
}
