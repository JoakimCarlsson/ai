package groq

import (
	"testing"

	openaisdk "github.com/openai/openai-go/v3"
)

// TestCompoundPreparedParamsStopSequencesArray verifies that the Groq compound
// client sends every provided stop sequence as an array, not just the first.
func TestCompoundPreparedParamsStopSequencesArray(t *testing.T) {
	c := &compoundClient{options: CompoundOptions{
		stopSequences: []string{"END", "STOP", "HALT"},
	}}

	params := c.preparedParams(
		[]openaisdk.ChatCompletionMessageParamUnion{},
		nil,
	)

	if params.Stop.OfString.Valid() {
		t.Fatalf(
			"expected OfString to be unset, got %q",
			params.Stop.OfString.Value,
		)
	}
	got := params.Stop.OfStringArray
	want := []string{"END", "STOP", "HALT"}
	if len(got) != len(want) {
		t.Fatalf(
			"expected %d stop sequences, got %d: %v",
			len(want),
			len(got),
			got,
		)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stop[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestCompoundPreparedParamsStopSequencesCappedAtFour verifies the Groq stop
// limit of 4 is enforced, matching OpenAI.
func TestCompoundPreparedParamsStopSequencesCappedAtFour(t *testing.T) {
	c := &compoundClient{options: CompoundOptions{
		stopSequences: []string{"1", "2", "3", "4", "5", "6"},
	}}

	params := c.preparedParams(
		[]openaisdk.ChatCompletionMessageParamUnion{},
		nil,
	)

	if len(params.Stop.OfStringArray) != 4 {
		t.Fatalf(
			"expected stop sequences capped at 4, got %d: %v",
			len(params.Stop.OfStringArray),
			params.Stop.OfStringArray,
		)
	}
}
