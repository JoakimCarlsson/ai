package decision

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Answer is the typed answer to one [Question]. The set is closed: an Answer
// is a [ChoiceAnswer], [ScoreAnswer] or [NoulAnswer], matching its question.
type Answer interface {
	// Type returns the answer kind: "choice", "score" or "noul".
	Type() string
}

// ChoiceAnswer answers a [ChoiceQuestion].
type ChoiceAnswer struct {
	// Choice is the name of the most probable option.
	Choice string `json:"choice"`
	// Probabilities maps each option name to its probability.
	Probabilities map[string]float64 `json:"probabilities"`
	// Confidence is the model's confidence in the answer, from 0 to 1.
	Confidence float64 `json:"confidence"`
}

// Type returns "choice".
func (ChoiceAnswer) Type() string { return "choice" }

// ScoreAnswer answers a [ScoreQuestion].
type ScoreAnswer struct {
	// Score is the probability-weighted level; it may fall between levels.
	Score float64 `json:"score"`
	// Probabilities holds the probability of each level, indexed by level.
	Probabilities []float64 `json:"probabilities"`
	// Legend echoes the level descriptions of the question, indexed by level.
	Legend []string `json:"legend"`
	// Confidence is the model's confidence in the answer, from 0 to 1.
	Confidence float64 `json:"confidence"`
}

// Type returns "score".
func (ScoreAnswer) Type() string { return "score" }

// UnmarshalJSON decodes a score answer whose probabilities and legend are
// objects keyed by level ("0", "1", ...) into slices indexed by level.
func (a *ScoreAnswer) UnmarshalJSON(data []byte) error {
	var raw struct {
		Score         float64            `json:"score"`
		Probabilities map[string]float64 `json:"probabilities"`
		Legend        map[string]string  `json:"legend"`
		Confidence    float64            `json:"confidence"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	probs, err := byLevel(raw.Probabilities)
	if err != nil {
		return err
	}
	legend, err := byLevel(raw.Legend)
	if err != nil {
		return err
	}
	*a = ScoreAnswer{
		Score:         raw.Score,
		Probabilities: probs,
		Legend:        legend,
		Confidence:    raw.Confidence,
	}
	return nil
}

func byLevel[T any](m map[string]T) ([]T, error) {
	out := make([]T, len(m))
	for k, v := range m {
		i, err := strconv.Atoi(k)
		if err != nil || i < 0 || i >= len(out) {
			return nil, fmt.Errorf("invalid score level %q", k)
		}
		out[i] = v
	}
	return out, nil
}

// NoulAnswer answers a [NoulQuestion].
type NoulAnswer struct {
	// Probability is the probability of "yes", from 0 to 1.
	Probability float64 `json:"noul"`
}

// Type returns "noul".
func (NoulAnswer) Type() string { return "noul" }

// Usage tracks the resource consumption of a decision request.
type Usage struct {
	// InputTokens is the number of billable input tokens.
	InputTokens int64 `json:"input_tokens"`
	// OutputTokens is the number of output tokens.
	OutputTokens int64 `json:"output_tokens"`
}

// Cost returns the cost of the usage under the given model's pricing, in the
// model's currency.
func (u Usage) Cost(m DecisionModel) float64 {
	return float64(u.InputTokens)/1e6*m.CostPer1MInputTokens +
		float64(u.OutputTokens)/1e6*m.CostPer1MOutputTokens
}

// Response holds the answers to a [Decider.Decide] call.
type Response struct {
	// Answers maps each question name to its typed answer.
	Answers map[string]Answer
	// Usage tracks resource consumption for this request.
	Usage Usage
	// Model identifies which model answered.
	Model string
}

// Choice returns the answer to the named question as a [ChoiceAnswer].
// ok is false if there is no such answer or it is of another type.
func (r *Response) Choice(name string) (a ChoiceAnswer, ok bool) {
	a, ok = r.Answers[name].(ChoiceAnswer)
	return a, ok
}

// Score returns the answer to the named question as a [ScoreAnswer].
// ok is false if there is no such answer or it is of another type.
func (r *Response) Score(name string) (a ScoreAnswer, ok bool) {
	a, ok = r.Answers[name].(ScoreAnswer)
	return a, ok
}

// Noul returns the answer to the named question as a [NoulAnswer].
// ok is false if there is no such answer or it is of another type.
func (r *Response) Noul(name string) (a NoulAnswer, ok bool) {
	a, ok = r.Answers[name].(NoulAnswer)
	return a, ok
}

// UnmarshalAnswer decodes one answer object, dispatching on its "type" field.
// Vendor packages use it to decode the answers of a response.
func UnmarshalAnswer(data []byte) (Answer, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, err
	}
	switch head.Type {
	case "choice":
		var a ChoiceAnswer
		err := json.Unmarshal(data, &a)
		return a, err
	case "score":
		var a ScoreAnswer
		err := json.Unmarshal(data, &a)
		return a, err
	case "noul":
		var a NoulAnswer
		err := json.Unmarshal(data, &a)
		return a, err
	default:
		return nil, fmt.Errorf("unsupported answer type %q", head.Type)
	}
}
