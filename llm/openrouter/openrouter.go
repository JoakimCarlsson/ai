// Package openrouter provides an OpenAI-compatible LLM client targeting OpenRouter.
//
// OpenRouter is a routing layer over many providers; it speaks the OpenAI
// chat-completions wire format, so this package is a thin wrapper over
// [llm/openai] that fixes the base URL to https://openrouter.ai/api/v1.
//
// The chat client preserves each response's complete reasoning_details array,
// including signatures and encrypted data, in message history and replays it
// unchanged on later assistant turns. [NewResponsesLLM] targets the stateless
// Responses API with encrypted reasoning replay.
//
// OpenRouter exposes far more models than the [model] package catalogues; pass
// any OpenRouter model id via [llmopenai.WithModel] even without a registered
// entry in [model].
package openrouter

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical OpenRouter API endpoint.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs an OpenRouter LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override (e.g. to point at a regional endpoint).
func NewLLM(opts ...Option) llm.LLM {
	return llmopenai.NewLLM(
		append([]Option{
			llmopenai.WithBaseURL(DefaultBaseURL),
			llmopenai.WithReasoningDetailsReplay(true),
		}, opts...)...)
}

// NewResponsesLLM constructs an OpenRouter client targeting the stateless
// Responses API, reusing the OpenAI Responses implementation with OpenRouter's
// endpoint. Storage defaults to false, which requests encrypted reasoning for
// models with CanReason and replays it from message history; previous_response_id
// is never sent. OpenRouter's Responses API is stateless-only and rejects
// store=true with an API error, so do not pass
// [llmopenai.WithResponsesStore] with true. Pass the API key with
// [llmopenai.WithResponsesAPIKey]; other options may override the defaults.
func NewResponsesLLM(opts ...llmopenai.ResponsesOption) llm.LLM {
	return llmopenai.NewResponsesLLM(
		append([]llmopenai.ResponsesOption{
			llmopenai.WithResponsesBaseURL(DefaultBaseURL),
			llmopenai.WithResponsesStore(false),
		}, opts...)...)
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
	return llmopenai.WithRequestJSONField("provider", provider)
}

// WithModelFallbacks sets OpenRouter's models fallback array. When the primary
// model (set via [llmopenai.WithModel]) errors or is unavailable, OpenRouter
// automatically retries the next model in this list. See
// https://openrouter.ai/docs/features/model-routing.
func WithModelFallbacks(models ...string) Option {
	return llmopenai.WithRequestJSONField("models", models)
}

// WithTopK limits token sampling to the top K candidates. Re-exported from
// [llmopenai.WithTopK]; OpenRouter honors top_k for providers that support it.
func WithTopK(k int64) Option {
	return llmopenai.WithTopK(k)
}
