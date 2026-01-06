package agent

import (
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
)

type ChatResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	ToolResults  []ToolExecutionResult
	Usage        llm.TokenUsage
	FinishReason message.FinishReason
}

type ToolExecutionResult struct {
	ToolCallID string
	ToolName   string
	Input      string
	Output     string
	IsError    bool
}

type ChatEvent struct {
	Type       types.EventType
	Content    string
	Thinking   string
	ToolCall   *message.ToolCall
	ToolResult *ToolExecutionResult
	Response   *ChatResponse
	Error      error
}
