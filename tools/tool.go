// Package tools carries what a provider's hosted tools cost.
//
// A hosted tool -- web search, file search -- is billed separately from the
// model that called it, per invocation rather than per token, at a rate every
// model pays alike. A provider's own note on the web-search rate reads
// "+ Search content tokens billed at model rates", so a call that searched is
// billed twice over: once here, per call, and once through the model's own
// catalog for the text it read.
package tools

// Tool is one hosted tool, with what a provider charges to invoke it. The
// rates are a default, not an authority; an operator-supplied rate should
// win.
type Tool struct {
	// ID is the catalog key, as the provider names the tool.
	ID string
	// Name is the human-readable name, for a chooser or a report.
	Name string
	// Provider is the service this tool is served by.
	Provider string
	// APIModel is the id the provider's own API uses, which is not always the
	// catalog key.
	APIModel string

	// Currency is the ISO 4217 code the cost fields are denominated in.
	Currency string

	// CostPer1KCalls is the cost of one thousand invocations, in Currency --
	// the unit the provider publishes rates in. Zero means the source
	// publishes no per-call rate for this tool, not that invoking it is
	// free; see CallRate.
	CostPer1KCalls float64
}

// CallRate returns what one thousand invocations cost, and whether the
// catalog publishes a rate at all -- a missing rate handed back as a bare
// zero would be spend recorded as free.
func (t Tool) CallRate() (float64, bool) {
	if t.CostPer1KCalls <= 0 {
		return 0, false
	}
	return t.CostPer1KCalls, true
}
