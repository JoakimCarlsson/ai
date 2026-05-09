package voice

import (
	"time"

	"github.com/joakimcarlsson/ai/message"
)

// EventType identifies the kind of event emitted on Conversation.Events.
type EventType string

// EventType values.
const (
	EventReady                 EventType = "ready"
	EventUserTranscriptPartial EventType = "user_transcript_partial"
	EventUserTranscriptFinal   EventType = "user_transcript_final"
	EventAssistantDelta        EventType = "assistant_delta"
	EventAssistantDone         EventType = "assistant_done"
	EventToolCallStart         EventType = "tool_call_start"
	EventToolCallEnd           EventType = "tool_call_end"
	EventTTSStarted            EventType = "tts_started"
	EventTTSEnded              EventType = "tts_ended"
	EventConversationEnd       EventType = "conversation_end"
	EventError                 EventType = "error"
)

// Event is a single observation emitted during a conversation. Fields are
// populated based on Type.
type Event struct {
	Type       EventType
	Timestamp  time.Time
	Text       string               // transcripts, assistant_delta
	ToolCall   *message.ToolCall    // tool_call_start
	ToolResult *ToolExecutionResult // tool_call_end
	Error      error                // error
}

// ToolExecutionResult captures the outcome of a single tool invocation.
type ToolExecutionResult struct {
	ToolCallID string
	ToolName   string
	Input      string
	Output     string
	IsError    bool
	Duration   time.Duration
}
