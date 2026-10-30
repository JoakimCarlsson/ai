// Package openrouter provides a [decision.Decider] that reaches TypeSafe's Jev
// (System One) through OpenRouter.
//
// OpenRouter serves the System One API at POST /api/v1/systemone with the same
// wire format as TypeSafe, billed to an OpenRouter key, so this package is a
// thin wrapper over [decision/jev] that fixes the base URL, the API key
// environment variable and the model catalog.
//
// OpenRouter lists decision models from several vendors; this package has only
// been exercised with Jev. Pass any other id with [WithModelID]. OpenRouter also
// has an alpha /api/alpha/decisions endpoint, which is not used here.
package openrouter

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

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
// [decision.Decider] is wrapped with [decision.WithTracing] and is also a
// [decision.ModelLister], which lists the decision models OpenRouter serves.
// Responses carry OpenRouter's billed cost in [decision.Usage.ReportedCost].
func NewDecider(opts ...Option) decision.Decider {
	return decision.WithTracing(jev.NewClient(append([]Option{
		jev.WithBaseURL(DefaultBaseURL),
		jev.WithAPIKey(os.Getenv(apiKeyEnv)),
		jev.WithModel(Models[JevLatest]),
		jev.WithModelsEndpoint(
			"/v1/models?output_modalities=decisions",
			decodeModels,
		),
	}, opts...)...))
}

func decodeModels(body []byte) ([]decision.ModelInfo, error) {
	var r struct {
		Data []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Created     int64  `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("failed to decode models: %w", err)
	}
	out := make([]decision.ModelInfo, 0, len(r.Data))
	for _, m := range r.Data {
		info := decision.ModelInfo{Name: m.ID, Description: m.Description}
		if m.Created > 0 {
			info.ReleaseDate = time.Unix(m.Created, 0).UTC()
		}
		out = append(out, info)
	}
	return out, nil
}
