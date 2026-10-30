// Package openrouter provides a [decision.Decider] that reaches TypeSafe's Jev
// (System One) through OpenRouter.
//
// OpenRouter serves the System One API at POST /api/v1/systemone with the same
// wire format as TypeSafe, billed to an OpenRouter key, so this package is a
// thin wrapper over [decision/jev] that fixes the base URL, the API key
// environment variable and the model catalog.
//
// OpenRouter also has an alpha /api/alpha/decisions endpoint; it is not used
// here.
package openrouter

import (
	"os"

	"github.com/joakimcarlsson/ai/decision"
	"github.com/joakimcarlsson/ai/decision/jev"
)

const (
	// DefaultBaseURL is the OpenRouter API root, without the /v1/systemone
	// path.
	DefaultBaseURL = "https://openrouter.ai/api"
	apiKeyEnv      = "OPENROUTER_API_KEY"
)

// Option re-exports [jev.Option] for caller convenience.
type Option = jev.Option

// WithModelID selects a model by raw OpenRouter id, for models [Models] does
// not catalogue. Nothing is validated locally and the cost fields are zero.
func WithModelID(id string) Option {
	return jev.WithModel(decision.Model{
		ID:       id,
		Name:     id,
		Provider: "openrouter",
		APIModel: id,
	})
}

// NewDecider constructs a Jev decider client that talks to OpenRouter. The
// API key defaults to the OPENROUTER_API_KEY environment variable and the
// model to Models[JevLatest]; pass options to override either. The returned
// [decision.Decider] is wrapped with [decision.WithTracing].
func NewDecider(opts ...Option) decision.Decider {
	return jev.NewDecider(append([]Option{
		jev.WithBaseURL(DefaultBaseURL),
		jev.WithAPIKey(os.Getenv(apiKeyEnv)),
		jev.WithModel(Models[JevLatest]),
	}, opts...)...)
}
