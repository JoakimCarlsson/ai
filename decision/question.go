package decision

import "encoding/json"

// Question is a typed question asked of a [Decider] about a state. The set is
// closed: build one with [Choice], [Score] or [Noul].
type Question interface {
	json.Marshaler
	// Type returns the question kind: "choice", "score" or "noul".
	Type() string
}

// ChoiceQuestion asks which one of several named options fits the state.
type ChoiceQuestion struct {
	// Instructions describes what is being decided.
	Instructions string `json:"instructions,omitempty"`
	// Criteria maps each option name to a description of when it applies.
	Criteria map[string]string `json:"criteria"`
}

// Type returns "choice".
func (ChoiceQuestion) Type() string { return "choice" }

// MarshalJSON encodes the question in the wire format with its type tag.
func (q ChoiceQuestion) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type         string            `json:"type"`
		Instructions string            `json:"instructions,omitempty"`
		Criteria     map[string]string `json:"criteria"`
	}{q.Type(), q.Instructions, q.Criteria})
}

// ScoreQuestion asks where the state falls on an ordered scale.
type ScoreQuestion struct {
	// Instructions describes what is being scored.
	Instructions string `json:"instructions,omitempty"`
	// Criteria describes each level of the scale; position sets the score,
	// starting at 0.
	Criteria []string `json:"criteria"`
}

// Type returns "score".
func (ScoreQuestion) Type() string { return "score" }

// MarshalJSON encodes the question in the wire format with its type tag.
func (q ScoreQuestion) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type         string   `json:"type"`
		Instructions string   `json:"instructions,omitempty"`
		Criteria     []string `json:"criteria"`
	}{q.Type(), q.Instructions, q.Criteria})
}

// NoulQuestion asks a yes/no question about the state.
type NoulQuestion struct {
	// Instructions is the statement or question to judge.
	Instructions string `json:"instructions,omitempty"`
}

// Type returns "noul".
func (NoulQuestion) Type() string { return "noul" }

// MarshalJSON encodes the question in the wire format with its type tag.
func (q NoulQuestion) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type         string `json:"type"`
		Instructions string `json:"instructions,omitempty"`
	}{q.Type(), q.Instructions})
}

// Choice builds a question that picks one option from criteria, a map of
// option name to description.
func Choice(instructions string, criteria map[string]string) Question {
	return ChoiceQuestion{Instructions: instructions, Criteria: criteria}
}

// Score builds a question that rates the state on a scale whose levels are
// described by levels, lowest first.
func Score(instructions string, levels ...string) Question {
	return ScoreQuestion{Instructions: instructions, Criteria: levels}
}

// Noul builds a yes/no question answered as a probability from 0 to 1.
func Noul(instructions string) Question {
	return NoulQuestion{Instructions: instructions}
}
