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
// A few behavioural differences from OpenAI proper are worth knowing:
//
//   - response_format defaults to pcm, not mp3. Pass
//     [ttsopenai.WithOutputFormat]("mp3") for MP3; the only two documented
//     values are "mp3" and "pcm".
//   - speed is honored only by upstreams that support it (OpenAI does) and is
//     silently ignored by the rest.
//   - voice is sent only when set with [ttsopenai.WithVoice], since most
//     upstreams reject OpenAI's alloy default and some (Seed Audio, HeyGen,
//     Fish, Speechify) need no voice at all.
//   - [ttsopenai.WithInstructions] is the documented way to steer delivery
//     on Gemini TTS, Fish Audio and gpt-4o-mini-tts.
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

// WithModelID selects a speech model by raw OpenRouter id, for the models
// [Models] does not catalogue. It is shorthand for
// [ttsopenai.WithModel] with a bare [tts.AudioModel], and it is the intended
// path for anything OpenRouter has routed since this package was last updated:
//
//	ttsopenrouter.WithModelID("minimax/speech-2.8-hd")
//
// Nothing is validated locally, so the capability and per-1M-character cost
// fields a registered [tts.AudioModel] carries are zero.
func WithModelID(id string) Option {
	return ttsopenai.WithModel(tts.AudioModel{
		APIModel: id,
		Provider: "openrouter",
	})
}

// WithProviderRouting sets OpenRouter's provider routing object, which the
// speech endpoint documents alongside model, input, voice, response_format and
// speed. order lists provider slugs to try in preference order; allowFallbacks
// controls whether OpenRouter may fall back to providers outside that list when
// they are unavailable. See
// https://openrouter.ai/docs/features/provider-routing.
//
// There is deliberately no WithModelFallbacks counterpart here. OpenRouter
// documents the models fallback array for chat completions and /api/v1/messages
// only, not for the audio endpoints, and the request schemas ignore fields they
// do not know — so sending models would look like it worked while doing
// nothing. Fall back in caller code by constructing a second client instead.
func WithProviderRouting(order []string, allowFallbacks bool) Option {
	return WithProviderPreferences(ProviderPreferences{
		Order:          order,
		AllowFallbacks: &allowFallbacks,
	})
}

// ProviderPreferences is OpenRouter's provider routing object. Unset fields
// are left out of the request, so OpenRouter's defaults apply to them.
type ProviderPreferences struct {
	// Order lists provider slugs to try, in preference order.
	Order []string `json:"order,omitempty"`
	// Only restricts routing to these provider slugs, such as one known to
	// support the chosen voice or format.
	Only []string `json:"only,omitempty"`
	// Ignore excludes these provider slugs.
	Ignore []string `json:"ignore,omitempty"`
	// AllowFallbacks lets OpenRouter use providers outside Order when those
	// are unavailable.
	AllowFallbacks *bool `json:"allow_fallbacks,omitempty"`
	// ZDR restricts routing to providers with zero data retention.
	ZDR *bool `json:"zdr,omitempty"`
	// DataCollection is "allow" or "deny": whether providers that may store
	// or train on the data are used.
	DataCollection string `json:"data_collection,omitempty"`
}

// WithProviderPreferences sets the whole provider routing object. It replaces
// anything [WithProviderRouting] set, and the other way round. See
// https://openrouter.ai/docs/features/provider-routing.
func WithProviderPreferences(p ProviderPreferences) Option {
	return ttsopenai.WithRequestJSONField("provider", p)
}
