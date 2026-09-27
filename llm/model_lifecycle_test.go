package llm

import "testing"

// TestDeprecatedReadsTheProvidersOwnWord confirms Deprecated reports true
// only for the provider's own "deprecated" state, treating an unpublished
// state the same as "active" rather than as a claim either way.
func TestDeprecatedReadsTheProvidersOwnWord(t *testing.T) {
	for _, tc := range []struct {
		state string
		want  bool
	}{
		{"deprecated", true},
		{"active", false},
		{"", false},
	} {
		if got := (Model{State: tc.state}).Deprecated(); got != tc.want {
			t.Errorf(
				"State %q: Deprecated() = %v, want %v",
				tc.state,
				got,
				tc.want,
			)
		}
	}
}
