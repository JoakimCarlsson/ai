// Package openrouter provides an OpenAI-compatible TTS client targeting
// OpenRouter.
//
// OpenRouter is a routing layer over many providers. Its POST
// /api/v1/audio/speech endpoint takes the OpenAI Audio Speech request shape
// (model, input, voice, response_format, speed) and answers with raw audio
// bytes, so this package is a thin wrapper over [tts/openai] that fixes the
// base URL to https://openrouter.ai/api/v1.
//
// OpenRouter exposes far more speech models than the [model] package
// catalogues; pass any OpenRouter speech model id via [ttsopenai.WithModel]
// even without a registered entry in [model].
//
// Two behavioural differences from OpenAI proper are worth knowing:
//
//   - response_format defaults to pcm, not mp3. Pass
//     [ttsopenai.WithOutputFormat]("mp3") for MP3; the only two documented
//     values are "mp3" and "pcm".
//   - speed is honored only by upstreams that support it (OpenAI does) and is
//     silently ignored by the rest.
package openrouter

import (
	"github.com/joakimcarlsson/ai/tts"
	ttsopenai "github.com/joakimcarlsson/ai/tts/openai"
)

// DefaultBaseURL is the canonical OpenRouter API endpoint.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// Option re-exports [ttsopenai.Option] for caller convenience.
type Option = ttsopenai.Option

// NewGeneration constructs an OpenRouter TTS client.
//
// [ttsopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override (e.g. to point at a proxy).
func NewGeneration(opts ...Option) tts.Generation {
	return ttsopenai.NewGeneration(
		append([]Option{ttsopenai.WithBaseURL(DefaultBaseURL)}, opts...)...)
}

// WithProviderRouting sets OpenRouter's provider routing object. order lists
// provider slugs to try in preference order; allowFallbacks controls whether
// OpenRouter may fall back to providers outside that list when they are
// unavailable. See https://openrouter.ai/docs/features/provider-routing.
func WithProviderRouting(order []string, allowFallbacks bool) Option {
	provider := map[string]any{"allow_fallbacks": allowFallbacks}
	if len(order) > 0 {
		provider["order"] = order
	}
	return ttsopenai.WithRequestJSONField("provider", provider)
}

// WithModelFallbacks sets OpenRouter's models fallback array. When the primary
// model (set via [ttsopenai.WithModel]) errors or is unavailable, OpenRouter
// automatically retries the next model in this list. See
// https://openrouter.ai/docs/features/model-routing.
func WithModelFallbacks(models ...string) Option {
	return ttsopenai.WithRequestJSONField("models", models)
}
