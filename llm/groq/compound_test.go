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
