package decision

import (
	"context"
	"time"
)

// ModelInfo describes a model a provider offers to the authenticated account.
type ModelInfo struct {
	// Name is the identifier to pass as the model of a request.
	Name string
	// Description summarises the model.
	Description string
	// ReleaseDate is when the model was released; zero if unknown.
	ReleaseDate time.Time
}

// ModelLister is implemented by a [Decider] whose provider can list its
// models. Check for it with a type assertion:
//
//	if l, ok := decider.(decision.ModelLister); ok {
//		models, err := l.ListModels(ctx)
//	}
type ModelLister interface {
	// ListModels returns the models and aliases available to the account.
	ListModels(ctx context.Context) ([]ModelInfo, error)
}
