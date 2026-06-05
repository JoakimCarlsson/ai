// Package eval provides a small, type-safe evaluation harness.
//
// The framework is generic over two type parameters:
//
//   - E is the per-case "Extras" type carrying domain-specific
//     expectations (ExpectedDocIDs for RAG, expected tool calls for
//     agents, role-play criteria for prompts, etc.).
//   - O is the per-output Extras type carrying domain-specific
//     observations (retrieved chunks, tool calls fired, etc.).
//
// Subject is a generic interface; metrics are typed over the same
// E and O so the runner enforces compatibility at compile time.
// LLM-as-judge metrics live in eval/judge and accept extractor
// functions instead of consulting an untyped map.
package eval

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Case is one row of an eval set parameterised by the per-case
// extras type E.
type Case[E any] struct {
	ID       string
	Input    string
	Expected string
	Extras   E
}

// Output is the result of running one Subject against one Case.
type Output[O any] struct {
	Text    string
	Latency time.Duration
	Extras  O
}

// Subject is the unit under evaluation. It produces an Output[O]
// for one input string.
type Subject[O any] interface {
	Run(ctx context.Context, input string) (Output[O], error)
}

// SubjectName is implemented by Subjects that want to label
// themselves in the produced Report. Optional.
type SubjectName interface {
	SubjectName() string
}

// Score is one metric's verdict on one case. The framework keeps
// Score concrete; metric-specific structured detail belongs on the
// metric's own typed return shape if the metric author wants it,
// surfaced through Reason.
type Score struct {
	Value  float64
	Pass   bool
	Reason string
}

// Metric scores one (Case[E], Output[O]) pair.
type Metric[E, O any] interface {
	Name() string
	Score(ctx context.Context, c Case[E], out Output[O]) (Score, error)
}

// CaseResult bundles one case's output and per-metric scores.
type CaseResult[E, O any] struct {
	Case       Case[E]
	Output     Output[O]
	RunErr     error
	Scores     map[string]Score
	MetricErrs map[string]error
}

// Report is the full output of Run.
type Report[E, O any] struct {
	Subject    string
	Cases      []CaseResult[E, O]
	Aggregates map[string]Aggregate
	Started    time.Time
	Finished   time.Time
}

// Aggregate is the per-metric summary across all cases.
type Aggregate struct {
	Mean       float64
	PassRate   float64
	Successes  int
	Failures   int
	NumScored  int
	NumErrored int
}

// Run evaluates subject against each case and scores every output
// against every metric. Errors from Subject.Run or Metric.Score are
// captured per case, never returned at the top level, so a single
// bad case does not abort the whole run.
//
// The runner is sequential. Callers parallelise by partitioning
// cases and merging Reports.
func Run[E, O any](
	ctx context.Context,
	subject Subject[O],
	cases []Case[E],
	metrics []Metric[E, O],
) Report[E, O] {
	rep := Report[E, O]{
		Started: time.Now(),
		Cases:   make([]CaseResult[E, O], 0, len(cases)),
	}
	if s, ok := subject.(SubjectName); ok {
		rep.Subject = s.SubjectName()
	}

	for _, c := range cases {
		cr := CaseResult[E, O]{
			Case:       c,
			Scores:     make(map[string]Score),
			MetricErrs: make(map[string]error),
		}
		t0 := time.Now()
		out, err := subject.Run(ctx, c.Input)
		out.Latency = time.Since(t0)
		cr.Output = out
		if err != nil {
			cr.RunErr = err
			rep.Cases = append(rep.Cases, cr)
			continue
		}

		for _, m := range metrics {
			s, err := m.Score(ctx, c, out)
			if err != nil {
				cr.MetricErrs[m.Name()] = err
				continue
			}
			cr.Scores[m.Name()] = s
		}
		rep.Cases = append(rep.Cases, cr)
	}

	rep.Finished = time.Now()
	rep.Aggregates = aggregate[E, O](rep.Cases, metrics)
	return rep
}

func aggregate[E, O any](
	crs []CaseResult[E, O],
	metrics []Metric[E, O],
) map[string]Aggregate {
	out := map[string]Aggregate{}
	for _, m := range metrics {
		var (
			sum, passes float64
			scored      int
			errs        int
		)
		for _, cr := range crs {
			if cr.RunErr != nil {
				errs++
				continue
			}
			if _, hadErr := cr.MetricErrs[m.Name()]; hadErr {
				errs++
				continue
			}
			s, ok := cr.Scores[m.Name()]
			if !ok {
				continue
			}
			sum += s.Value
			if s.Pass {
				passes++
			}
			scored++
		}
		agg := Aggregate{NumScored: scored, NumErrored: errs}
		if scored > 0 {
			agg.Mean = sum / float64(scored)
			agg.PassRate = passes / float64(scored)
			agg.Successes = int(passes)
			agg.Failures = scored - agg.Successes
		}
		out[m.Name()] = agg
	}
	return out
}

// FormatReport renders r as a human-readable text report.
func FormatReport[E, O any](r Report[E, O]) string {
	var b strings.Builder
	if r.Subject != "" {
		fmt.Fprintf(&b, "Subject: %s\n", r.Subject)
	}
	fmt.Fprintf(&b, "Cases:   %d\n", len(r.Cases))
	fmt.Fprintf(&b, "Wall:    %s\n\n",
		r.Finished.Sub(r.Started).Round(time.Millisecond))

	names := make([]string, 0, len(r.Aggregates))
	for name := range r.Aggregates {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(&b, "%-28s  %8s  %8s  %8s  %8s\n",
		"metric", "mean", "pass%", "scored", "errors")
	for _, name := range names {
		a := r.Aggregates[name]
		fmt.Fprintf(&b, "%-28s  %8.3f  %7.0f%%  %8d  %8d\n",
			name, a.Mean, 100*a.PassRate, a.NumScored, a.NumErrored)
	}

	fmt.Fprintln(&b, "\nPer-case:")
	for _, cr := range r.Cases {
		fmt.Fprintf(&b, "  [%s] (%s)\n", cr.Case.ID,
			cr.Output.Latency.Round(time.Millisecond))
		if cr.RunErr != nil {
			fmt.Fprintf(&b, "    RUN ERROR: %v\n", cr.RunErr)
			continue
		}
		for _, name := range names {
			if e, ok := cr.MetricErrs[name]; ok {
				fmt.Fprintf(&b, "    %-26s ERROR: %v\n", name, e)
				continue
			}
			s, ok := cr.Scores[name]
			if !ok {
				continue
			}
			pass := "FAIL"
			if s.Pass {
				pass = "PASS"
			}
			fmt.Fprintf(&b, "    %-26s %s  value=%.3f", name, pass, s.Value)
			if s.Reason != "" {
				fmt.Fprintf(&b, "  reason=%s", oneLine(s.Reason, 120))
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func oneLine(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
