package hostedtool

import "testing"

// TestAToolWithNoPublishedRateIsNotFree confirms CallRate reports an
// unpublished rate as such rather than as a bare zero.
func TestAToolWithNoPublishedRateIsNotFree(t *testing.T) {
	for _, tc := range []struct {
		name string
		tool Tool
	}{
		{"nothing published", Tool{ID: "web-search"}},
		{"an explicit zero", Tool{ID: "web-search", CostPer1KCalls: 0}},
		{
			"a negative, which is not a price",
			Tool{ID: "web-search", CostPer1KCalls: -1},
		},
	} {
		if rate, ok := tc.tool.CallRate(); ok {
			t.Errorf(
				"%s: CallRate() = (%v, true), want it reported as unpublished",
				tc.name,
				rate,
			)
		}
	}
}

// TestAPublishedRateIsReturnedInTheUnitItWasPublishedIn confirms the rate
// stays in its published per-1k-calls unit rather than being scaled.
func TestAPublishedRateIsReturnedInTheUnitItWasPublishedIn(t *testing.T) {
	tool := Tool{ID: "web-search", Currency: "USD", CostPer1KCalls: 10}
	rate, ok := tool.CallRate()
	if !ok {
		t.Fatal("CallRate() reported a published rate as missing")
	}
	if rate != 10 {
		t.Errorf(
			"CallRate() = %v, want 10 -- the unit is per thousand calls",
			rate,
		)
	}
}
