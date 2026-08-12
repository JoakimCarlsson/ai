// Package openrouter provides an OpenAI-compatible speech-to-text client
// targeting OpenRouter.
//
// OpenRouter is a routing layer over many providers. Its POST
// /api/v1/audio/transcriptions endpoint accepts OpenAI-style
// multipart/form-data alongside its own JSON body with base64 audio, so this
// package is a thin wrapper over [stt/openai] that fixes the base URL to
// https://openrouter.ai/api/v1 and takes the multipart path.
//
// OpenRouter exposes far more transcription models than the [model] package
// catalogues; pass any OpenRouter transcription model id via
// [sttopenai.WithModel] even without a registered entry in [model].
//
// Two divergences from OpenAI proper are handled here rather than left to
// surprise the caller:
//
//   - OpenRouter has no /audio/translations route at all (it answers 404), so
//     [Client.Translate] returns [ErrTranslationNotSupported] instead of
//     issuing a request that cannot succeed. Transcribe with a model whose
//     upstream translates, or use a native vendor package.
//   - response_format="verbose_json" — and with it the segments and words
//     arrays — is only accepted by the OpenAI-compatible upstreams (OpenAI,
//     Groq, Together). Other upstreams reject it with HTTP 400. [stt/openai]
//     asks for verbose_json by default, so pass
//     [stt.WithResponseFormat]("json") per call when routing to an upstream
//     outside that set.
package openrouter

import (
	"context"
	"errors"

	"github.com/joakimcarlsson/ai/stt"
	sttopenai "github.com/joakimcarlsson/ai/stt/openai"
)

// DefaultBaseURL is the canonical OpenRouter API endpoint.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// ErrTranslationNotSupported is returned by [Client.Translate]. OpenRouter
// publishes /audio/transcriptions and /audio/speech but no
// /audio/translations counterpart, so there is no endpoint to call.
var ErrTranslationNotSupported = errors.New(
	"stt/openrouter: OpenRouter exposes no /audio/translations endpoint",
)

// Option re-exports [sttopenai.Option] for caller convenience.
type Option = sttopenai.Option

// WithModelID selects a transcription model by raw OpenRouter id, for the models
// [Models] does not catalogue. It is shorthand for
// [sttopenai.WithModel] with a bare [stt.TranscriptionModel], and it is the
// intended path for anything OpenRouter has routed since this package was last
// updated:
//
//	sttopenrouter.WithModelID("nvidia/parakeet-tdt-0.6b-v3")
//
// Nothing is validated locally, so the capability and cost fields a registered
// [stt.TranscriptionModel] carries are zero. In particular, pass
// [stt.WithResponseFormat]("json") unless the id routes to an OpenAI-compatible
// upstream, since the default verbose_json draws an HTTP 400 elsewhere.
func WithModelID(id string) Option {
	return sttopenai.WithModel(stt.TranscriptionModel{
		APIModel: id,
		Provider: "openrouter",
	})
}

// Client wraps the [stt/openai] client to replace [Client.Translate] with a
// clear error. Every other method is the embedded client's.
type Client struct {
	stt.SpeechToText
}

// NewSpeechToText constructs an OpenRouter speech-to-text client.
//
// [sttopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override (e.g. to point at a proxy).
func NewSpeechToText(opts ...Option) stt.SpeechToText {
	return &Client{
		SpeechToText: sttopenai.NewSpeechToText(
			append([]Option{
				sttopenai.WithBaseURL(DefaultBaseURL),
			}, opts...)...),
	}
}

// Translate returns [ErrTranslationNotSupported]; OpenRouter has no
// translation endpoint.
func (c *Client) Translate(
	_ context.Context,
	_ []byte,
	_ ...stt.Option,
) (*stt.Response, error) {
	return nil, ErrTranslationNotSupported
}
