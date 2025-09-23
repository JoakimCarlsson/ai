// Package types defines common types and constants used across the AI client library.
//
// This package contains shared type definitions for events, streaming responses,
// and other common data structures that are used by multiple packages within
// the AI client library.
//
// The primary focus is on event types for streaming responses, which allow
// clients to handle different kinds of events during streaming interactions
// with AI models.
package types

// EventType defines the type of streaming response event from AI models.
type EventType string

const (
	// EventContentStart indicates the beginning of content generation.
	EventContentStart EventType = "content_start"
	// EventContentDelta indicates a partial content update during streaming.
	EventContentDelta EventType = "content_delta"
	// EventContentStop indicates the end of content generation.
	EventContentStop EventType = "content_stop"
	// EventToolUseStart indicates the beginning of a tool use request.
	EventToolUseStart EventType = "tool_use_start"
	// EventToolUseDelta indicates a partial tool use update during streaming.
	EventToolUseDelta EventType = "tool_use_delta"
	// EventToolUseStop indicates the end of a tool use request.
	EventToolUseStop EventType = "tool_use_stop"
	// EventThinkingDelta indicates reasoning content for models that support chain-of-thought.
	EventThinkingDelta EventType = "thinking_delta"
	// EventComplete indicates the streaming response has completed successfully.
	EventComplete EventType = "complete"
	// EventError indicates an error occurred during streaming.
	EventError EventType = "error"
	// EventWarning indicates a warning occurred during streaming.
	EventWarning EventType = "warning"
)
