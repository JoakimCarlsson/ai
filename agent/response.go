package agent

import (
	"time"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// ChatResponse represents the complete result of an agent Chat or ChatStream call.
// Usage is aggregated across all LLM round-trips in the agent loop, not just the final call.
type ChatResponse struct {
	// Content is the final text response from the agent.
	Content string
	// ToolCalls contains any pending tool calls from the final LLM response.
	ToolCalls []message.ToolCall
	// ToolResults contains the results of all tool executions during the conversation.
	ToolResults []ToolExecutionResult
	// Usage is the aggregated token usage across all LLM calls in the agent loop.
	Usage llm.TokenUsage
	// FinishReason indicates why the agent stopped (end_turn, max_tokens, tool_use, etc.).
	FinishReason message.FinishReason
	// AgentName is the name of the agent that produced this response, set when a handoff occurred.
	AgentName string
	// TotalToolCalls is the total number of tool invocations across all iterations.
	TotalToolCalls int
	// TotalDuration is the wall-clock time from Chat() entry to return.
	TotalDuration time.Duration
	// TotalTurns is the number of LLM round-trips (API calls) made during the conversation.
	TotalTurns int
}

// ToolExecutionResult captures the outcome of a single tool invocation.
type ToolExecutionResult struct {
	// ToolCallID is the unique identifier for this tool call, matching the LLM's request.
	ToolCallID string
	// ToolName is the name of the tool that was executed.
	ToolName string
	// Input is the raw JSON input that was passed to the tool.
	Input string
	// Output is the tool's text response.
	Output string
	// IsError indicates whether the tool execution resulted in an error.
	IsError bool
	// Duration is the wall-clock time the tool execution took.
	Duration time.Duration
}

// ChatEvent represents a single streaming event emitted during ChatStream.
type ChatEvent struct {
	// Type identifies the kind of event (content_delta, tool_use_start, complete, error, etc.).
	Type types.EventType
	// Content contains partial text for EventContentDelta events.
	Content string
	// Thinking contains chain-of-thought text for EventThinkingDelta events.
	Thinking string
	// ToolCall contains tool call information for tool use events.
	ToolCall *message.ToolCall
	// ToolResult contains the result of a tool execution.
	ToolResult *ToolExecutionResult
	// Response contains the final ChatResponse for EventComplete events.
	Response *ChatResponse
	// Error contains error details for EventError events.
	Error error
	// AgentName is set on EventHandoff events to indicate the target agent.
	AgentName string
	// ConfirmationRequest is set on EventConfirmationRequired events with the details of the pending request.
	ConfirmationRequest *tool.ConfirmationRequest
}
