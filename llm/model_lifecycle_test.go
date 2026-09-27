package llm

import "testing"

// TestDeprecatedReadsTheProvidersOwnWord confirms Deprecated reports true for
// every state a provider uses to wind a model down, and false for current,
// pre-release and unpublished states.
func TestDeprecatedReadsTheProvidersOwnWord(t *testing.T) {
	for _, tc := range []struct {
		state string
		want  bool
	}{
		{"deprecated", true},
		{"legacy", true},
		{"retired", true},
		{"shutdown", true},
		{"active", false},
		{"stable", false},
		{"production", false},
		{"preview", false},
		{"experimental", false},
		{"eval", false},
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
