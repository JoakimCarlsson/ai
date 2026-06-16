package eval

import (
	"context"
	"fmt"
	"strings"
)

// Contains is a deterministic metric that passes when Output.Text
// contains the substring s (case-insensitive). It is generic over
// E and O because it only reads Output.Text and ignores Extras.
func Contains[E, O any](s string) Metric[E, O] {
	return &containsMetric[E, O]{needle: strings.ToLower(s)}
}

type containsMetric[E, O any] struct{ needle string }

func (c *containsMetric[E, O]) Name() string {
	return "contains:" + c.needle
}

func (c *containsMetric[E, O]) Score(
	_ context.Context,
	_ Case[E],
	out Output[O],
) (Score, error) {
	hit := strings.Contains(strings.ToLower(out.Text), c.needle)
	return Score{
		Value:  boolToFloat(hit),
		Pass:   hit,
		Reason: fmt.Sprintf("looking for %q", c.needle),
	}, nil
}

// ExactMatch passes when Output.Text equals Case.Expected exactly
// (whitespace-trimmed, case-sensitive).
func ExactMatch[E, O any]() Metric[E, O] {
	return &exactMatchMetric[E, O]{}
}

type exactMatchMetric[E, O any] struct{}

func (e *exactMatchMetric[E, O]) Name() string { return "exact_match" }

func (e *exactMatchMetric[E, O]) Score(
	_ context.Context,
	c Case[E],
	out Output[O],
) (Score, error) {
	got := strings.TrimSpace(out.Text)
	want := strings.TrimSpace(c.Expected)
	hit := got == want
	return Score{
		Value: boolToFloat(hit),
		Pass:  hit,
	}, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
