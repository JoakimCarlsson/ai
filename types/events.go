package types

// EventType defines the type of streaming response event
type EventType string

const (
	EventContentStart  EventType = "content_start"
	EventContentDelta  EventType = "content_delta"
	EventContentStop   EventType = "content_stop"
	EventToolUseStart  EventType = "tool_use_start"
	EventToolUseDelta  EventType = "tool_use_delta"
	EventToolUseStop   EventType = "tool_use_stop"
	EventThinkingDelta EventType = "thinking_delta"
	EventComplete      EventType = "complete"
	EventError         EventType = "error"
	EventWarning       EventType = "warning"
)
