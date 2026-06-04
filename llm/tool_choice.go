package llm

import "errors"

// ToolChoiceMode controls whether/which tool the model may call.
type ToolChoiceMode int

// ToolChoiceMode values.
const (
	ToolChoiceAuto     ToolChoiceMode = iota // model decides (default)
	ToolChoiceNone                           // never call a tool
	ToolChoiceRequired                       // must call some tool
	ToolChoiceSpecific                       // must call ToolChoice.Name
)

// ToolChoice is a vendor-neutral description of how the model should use the
// tools supplied with a request. Vendor packages expose it through a
// WithToolChoice option and translate it to their provider's wire format.
type ToolChoice struct {
	// Mode selects the tool-calling discipline.
	Mode ToolChoiceMode
	// Name is the tool the model must call; required when Mode is
	// [ToolChoiceSpecific] and ignored otherwise.
	Name string
}

// ErrToolChoiceNameRequired indicates a [ToolChoiceSpecific] choice was made
// without a tool name, which would produce a malformed provider request.
var ErrToolChoiceNameRequired = errors.New(
	"llm: ToolChoiceSpecific requires a non-empty Name",
)

// Validate reports whether the tool choice is well-formed. A
// [ToolChoiceSpecific] choice with an empty Name is rejected.
func (tc ToolChoice) Validate() error {
	if tc.Mode == ToolChoiceSpecific && tc.Name == "" {
		return ErrToolChoiceNameRequired
	}
	return nil
}
