package main

import (
	"context"
	"slices"
	"strings"
)

// openRouterModel is the subset of https://openrouter.ai/api/v1/models this
// tool reads.
type openRouterModel struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ContextLen   int64  `json:"context_length"`
	Architecture struct {
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture"`
	Pricing struct {
		Prompt         string `json:"prompt"`
		Completion     string `json:"completion"`
		InputCacheRead string `json:"input_cache_read"`
		ImageOutput    string `json:"image_output"`
		Image          string `json:"image"`
	} `json:"pricing"`
	TopProvider struct {
		ContextLength       int64 `json:"context_length"`
		MaxCompletionTokens int64 `json:"max_completion_tokens"`
	} `json:"top_provider"`
	SupportedParameters []string `json:"supported_parameters"`
}

type openRouterList struct {
	Data []openRouterModel `json:"data"`
}

const openRouterBase = "https://openrouter.ai/api/v1/models"

// openRouter serves every catalog from one endpoint, filtered by output
// modality: the unfiltered list is the chat catalog, and image, speech and
// transcription each have their own query.
func openRouter() provider {
	queries := []struct {
		kind kind
		url  string
	}{
		{kindChat, openRouterBase},
		{kindImage, openRouterBase + "?output_modalities=image"},
		{kindSpeech, openRouterBase + "?output_modalities=speech"},
		{
			kindTranscription,
			openRouterBase + "?output_modalities=transcription",
		},
	}

	return provider{
		name: "openrouter",
		fetch: func(ctx context.Context) ([]model, error) {
			var models []model
			for _, q := range queries {
				var list openRouterList
				if err := fetchJSON(ctx, q.url, &list); err != nil {
					return nil, err
				}
				for _, m := range list.Data {
					models = append(models, openRouterEntry(q.kind, m))
				}
			}
			return models, nil
		},
		targets: []target{
			{
				kind:       kindChat,
				path:       "llm/openrouter/models.go",
				pkg:        "openrouter",
				importPath: "github.com/joakimcarlsson/ai/llm",
				typeExpr:   "llm.Model",
				source:     openRouterBase,
				idPrefix:   "openrouter.",
				order:      chatFields,
				doc: []string{
					"OpenRouter routes to upstream providers and passes their rates",
					"through; the figures here are OpenRouter's own, in USD per 1M tokens.",
					"",
					"DefaultMaxTokens is derived from the routed provider's completion cap,",
					"and is not published per model.",
				},
				defaults: map[string]string{"Provider": `"openrouter"`},
			},
			{
				kind:       kindImage,
				path:       "image/openrouter/models.go",
				pkg:        "openrouter",
				importPath: "github.com/joakimcarlsson/ai/image",
				typeExpr:   "image.GenerationModel",
				source:     openRouterBase + "?output_modalities=image",
				idPrefix:   "openrouter.",
				order:      imageFields,
				doc: []string{
					"Pricing is USD per image. The API publishes a single rate per model,",
					"so an entry's size and quality table is written only when the model",
					"is new to the catalog and is carried over from then on.",
					"",
					"Supported sizes, qualities and aspect ratios are not part of the",
					"models response and are carried over from the previous catalog.",
				},
				defaults: map[string]string{"Provider": `"openrouter"`},
			},
			{
				kind:       kindSpeech,
				path:       "tts/openrouter/models.go",
				pkg:        "openrouter",
				importPath: "github.com/joakimcarlsson/ai/tts",
				typeExpr:   "tts.AudioModel",
				source:     openRouterBase + "?output_modalities=speech",
				idPrefix:   "openrouter.",
				order:      speechFields,
				doc: []string{
					"CostPer1MChars is USD per 1M input characters. Audio formats, default",
					"format and latency are not part of the models response and are carried",
					"over from the previous catalog.",
				},
				defaults: map[string]string{"Provider": `"openrouter"`},
			},
			{
				kind:       kindTranscription,
				path:       "stt/openrouter/models.go",
				pkg:        "openrouter",
				importPath: "github.com/joakimcarlsson/ai/stt",
				typeExpr:   "stt.TranscriptionModel",
				source:     openRouterBase + "?output_modalities=transcription",
				idPrefix:   "openrouter.",
				order:      transcriptionFields,
				doc: []string{
					"OpenRouter prices transcription per audio minute, so CostPer1MIn holds",
					"the USD per-minute rate rather than a per-token one.",
					"",
					"File size limits, timestamp, diarization and translation support are",
					"not part of the models response and are carried over from the previous",
					"catalog.",
				},
				defaults: map[string]string{"Provider": `"openrouter"`},
			},
		},
	}
}

func openRouterEntry(k kind, m openRouterModel) model {
	seed := map[string]string{}
	fields := map[string]string{
		"Name":     quote(openRouterName(m.Name)),
		"Provider": `"openrouter"`,
		"APIModel": quote(m.ID),
		"Currency": `"USD"`,
	}

	switch k {
	case kindChat:
		window := m.TopProvider.ContextLength
		if window == 0 {
			window = m.ContextLen
		}
		fields["CostPer1MIn"] = perMillion(m.Pricing.Prompt)
		fields["CostPer1MOut"] = perMillion(m.Pricing.Completion)
		fields["CostPer1MInCached"] = perMillion(m.Pricing.InputCacheRead)
		fields["ContextWindow"] = integer(window)
		fields["DefaultMaxTokens"] = integer(
			defaultMaxTokens(m.TopProvider.MaxCompletionTokens, window),
		)
		fields["CanReason"] = boolean(
			slices.Contains(m.SupportedParameters, "reasoning"),
		)
		fields["SupportsAttachments"] = boolean(hasAny(
			m.Architecture.InputModalities,
			"image", "file", "audio", "video",
		))
		fields["SupportsStructuredOut"] = boolean(
			slices.Contains(m.SupportedParameters, "structured_outputs"),
		)
		fields["SupportsImageGeneration"] = boolean(
			slices.Contains(m.Architecture.OutputModalities, "image"),
		)
	case kindImage:
		if perImage := parseFloat(m.Pricing.Image); perImage > 0 {
			seed["Pricing"] = imagePricing(perImage)
		}
		fields["MaxPromptTokens"] = integer(m.ContextLen)
	case kindSpeech:
		fields["CostPer1MChars"] = perMillion(m.Pricing.Prompt)
	case kindTranscription:
		fields["CostPer1MIn"] = amount(parseFloat(m.Pricing.Prompt))
		fields["CostPer1MOut"] = amount(parseFloat(m.Pricing.Completion))
	}

	return model{kind: k, apiModel: m.ID, fields: fields, seed: seed}
}

// openRouterName drops the vendor prefix the API puts on display names, since
// the catalog states the routing provider itself.
func openRouterName(name string) string {
	if _, rest, ok := strings.Cut(name, ": "); ok {
		name = rest
	}
	return "OpenRouter – " + name
}

// defaultMaxTokens caps the routed provider's completion limit at a value that
// is usable as a default, falling back to a quarter of the context window when
// the provider publishes no limit.
func defaultMaxTokens(maxCompletion, window int64) int64 {
	if maxCompletion > 0 {
		return min(maxCompletion, 50000)
	}
	if window > 0 {
		return min(window/4, 8192)
	}
	return 8192
}

// imagePricing renders a flat per-image rate. Only models OpenRouter prices per
// image get one; the rest keep whatever size and quality table the catalog
// already holds, since the API's remaining image rates are per token.
func imagePricing(perImage float64) string {
	return "map[string]map[string]float64{\n\"default\": {\"default\": " +
		amount(perImage) + "},\n}"
}

func hasAny(have []string, want ...string) bool {
	for _, w := range want {
		if slices.Contains(have, w) {
			return true
		}
	}
	return false
}
