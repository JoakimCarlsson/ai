// Package judge provides LLM-as-judge metrics for the eval harness.
//
// Each judge issues one LLM call per case and parses a JSON
// verdict. Faithfulness and Correctness are pre-built; Generic
// lets callers ship custom rubrics with the same parsing
// infrastructure.
//
// Because the eval framework is generic over case- and
// output-extras, judges accept extractor functions to surface the
// fields they need (retrieved context, off-topic flag) instead of
// reading from an untyped map.
package judge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joakimcarlsson/ai/eval"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
)

// Verdict is the JSON shape every judge prompts for. A single shape
// means callers can layer custom prompts without re-implementing
// parsing.
type Verdict struct {
	Pass      bool     `json:"pass"`
	Score     float64  `json:"score"`
	Reasoning string   `json:"reasoning"`
	Issues    []string `json:"issues,omitempty"`
}

// Faithfulness scores whether every claim in Output.Text is
// supported by the retrieved context returned by extractContext.
// Off-topic cases (where isOffTopic returns true, if provided)
// score 1 only when the assistant declines.
//
// extractContext is required; pass a function returning "" if your
// pipeline has no retrieved context.
func Faithfulness[E, O any](
	client llm.LLM,
	extractContext func(eval.Output[O]) string,
	isOffTopic func(eval.Case[E]) bool,
) eval.Metric[E, O] {
	return &judgeMetric[E, O]{
		name:   "faithfulness",
		client: client,
		buildPrompt: func(c eval.Case[E], out eval.Output[O]) string {
			contextStr := extractContext(out)
			if contextStr == "" {
				contextStr = "(no retrieved context recorded)"
			}
			off := false
			if isOffTopic != nil {
				off = isOffTopic(c)
			}
			offHint := ""
			if off {
				offHint = "\n\nIS_OFF_TOPIC: true. Score 1 only if the assistant declined to answer; score 0 if the assistant invented facts."
			}
			return fmt.Sprintf(
				`You are a strict evaluator for a retrieval-augmented question-answering system.

Decide whether every factual claim in the assistant's answer is supported by the retrieved context. Reasonable paraphrasing is allowed; new facts that do not appear in the context are not.%s

Respond with JSON only, no prose, no code fences:
{"pass": bool, "score": 0..1, "reasoning": "one short sentence", "issues": ["unsupported claim", ...]}

QUESTION:
%s

RETRIEVED CONTEXT:
%s

ASSISTANT ANSWER:
%s`,
				offHint,
				c.Input,
				contextStr,
				out.Text,
			)
		},
	}
}

// Correctness scores whether Output.Text captures the substance of
// Case.Expected. When isOffTopic returns true for a case, score 1
// only if the assistant declined.
func Correctness[E, O any](
	client llm.LLM,
	isOffTopic func(eval.Case[E]) bool,
) eval.Metric[E, O] {
	return &judgeMetric[E, O]{
		name:   "correctness",
		client: client,
		buildPrompt: func(c eval.Case[E], out eval.Output[O]) string {
			off := false
			if isOffTopic != nil {
				off = isOffTopic(c)
			}
			offHint := ""
			if off {
				offHint = "\n\nIS_OFF_TOPIC: true. The expected behaviour is to decline. Score 1 if the assistant declined and made no factual claims; score 0 if it answered substantively."
			}
			return fmt.Sprintf(
				`You are a strict evaluator. Decide whether the assistant's answer matches the substance of the golden expected answer. Key facts and numbers must agree; reasonable paraphrasing is allowed.%s

Respond with JSON only, no prose, no code fences:
{"pass": bool, "score": 0..1, "reasoning": "one short sentence", "issues": ["missing fact", ...]}

QUESTION:
%s

GOLDEN ANSWER:
%s

ASSISTANT ANSWER:
%s`,
				offHint,
				c.Input,
				c.Expected,
				out.Text,
			)
		},
	}
}

// Generic lets callers supply a custom prompt builder. The returned
// metric still parses a Verdict from the LLM response and surfaces
// it as eval.Score.
func Generic[E, O any](
	name string,
	client llm.LLM,
	build func(eval.Case[E], eval.Output[O]) string,
) eval.Metric[E, O] {
	return &judgeMetric[E, O]{name: name, client: client, buildPrompt: build}
}

type judgeMetric[E, O any] struct {
	name        string
	client      llm.LLM
	buildPrompt func(eval.Case[E], eval.Output[O]) string
}

func (j *judgeMetric[E, O]) Name() string { return j.name }

func (j *judgeMetric[E, O]) Score(
	ctx context.Context,
	c eval.Case[E],
	out eval.Output[O],
) (eval.Score, error) {
	prompt := j.buildPrompt(c, out)
	resp, err := j.client.SendMessages(ctx, []message.Message{
		message.NewSystemMessage(
			"You are a strict evaluator. Output JSON only and follow the schema exactly.",
		),
		message.NewUserMessage(prompt),
	}, nil)
	if err != nil {
		return eval.Score{}, fmt.Errorf("judge %s: %w", j.name, err)
	}

	raw := stripCodeFence(strings.TrimSpace(resp.Content))
	var v Verdict
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return eval.Score{}, fmt.Errorf(
			"judge %s: parse verdict: %w; raw=%q", j.name, err, raw,
		)
	}
	if v.Score < 0 {
		v.Score = 0
	}
	if v.Score > 1 {
		v.Score = 1
	}
	reason := v.Reasoning
	if len(v.Issues) > 0 {
		reason = reason + " | issues: " + strings.Join(v.Issues, "; ")
	}
	return eval.Score{
		Value:  v.Score,
		Pass:   v.Pass,
		Reason: reason,
	}, nil
}

func stripCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
