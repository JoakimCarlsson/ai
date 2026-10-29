// Package decision provides a unified interface for decision models.
//
// A decision model does not generate text. It reads a state and returns typed
// answers (a choice, a score or a probability) in a single pass, which suits
// routing, tool-call gating, classification and guardrails inside agent loops.
// This package defines the [Decider] interface and the question and answer
// types that flow through it. Concrete vendor implementations live in
// subpackages (decision/jev); each subpackage exports its own NewDecider
// constructor that returns a tracing-wrapped client implementing the interface.
//
// Example usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/decision"
//		"github.com/joakimcarlsson/ai/decision/jev"
//	)
//
//	decider := jev.NewDecider(jev.WithAPIKey("your-api-key"))
//
//	resp, err := decider.Decide(ctx, "My invoice is wrong and I need it today.",
//		map[string]decision.Question{
//			"department": decision.Choice("Route the ticket.", map[string]string{
//				"billing":   "Payments and invoices",
//				"technical": "Bugs and outages",
//			}),
//			"frustration": decision.Score("How frustrated is the customer?",
//				"calm", "annoyed", "furious"),
//			"urgent": decision.Noul("Is the customer asking for urgent help?"),
//		})
//	dept, _ := resp.Choice("department")
package decision

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/tracing"
)

// Decider defines the interface for decision models.
type Decider interface {
	// Decide answers every question over the same state in a single call. The
	// state may be a string, or any value that marshals to a JSON object or
	// array. The response holds one typed answer per question name.
	Decide(
		ctx context.Context,
		state any,
		questions map[string]Question,
	) (*Response, error)
	// Model returns the decision model configuration being used.
	Model() DecisionModel
}

// WithTracing wraps a Decider so every call records OpenTelemetry spans and
// metrics.
func WithTracing(inner Decider) Decider {
	return &tracingDecider{inner: inner}
}

type tracingDecider struct {
	inner Decider
}

func (t *tracingDecider) Model() DecisionModel {
	return t.inner.Model()
}

func (t *tracingDecider) Decide(
	ctx context.Context,
	state any,
	questions map[string]Question,
) (*Response, error) {
	m := t.inner.Model()
	start := time.Now()
	ctx, span := tracing.StartSpan(
		ctx,
		"decide "+m.APIModel,
		tracing.AttrOperationName.String("decide"),
		tracing.AttrSystem.String(m.Provider),
		tracing.AttrRequestModel.String(m.APIModel),
	)
	defer span.End()
	span.SetAttributes(tracing.AttrResultCount.Int(len(questions)))

	resp, err := t.inner.Decide(ctx, state, questions)
	if err != nil {
		tracing.SetError(span, err)
		tracing.RecordMetrics(
			ctx,
			"decide",
			m.APIModel,
			m.Provider,
			time.Since(start),
			0,
			0,
			err,
		)
		return nil, err
	}

	tracing.SetResponseAttrs(span,
		tracing.AttrUsageInputTokens.Int64(resp.Usage.InputTokens),
		tracing.AttrUsageOutputTokens.Int64(resp.Usage.OutputTokens),
	)
	tracing.RecordMetrics(
		ctx,
		"decide",
		m.APIModel,
		m.Provider,
		time.Since(start),
		resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
		nil,
	)
	return resp, nil
}
