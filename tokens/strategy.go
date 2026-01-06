package tokens

import (
	"context"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// Strategy defines how to manage context when it exceeds the model's limit.
type Strategy interface {
	Fit(ctx context.Context, input StrategyInput) (*StrategyResult, error)
}

// StrategyResult contains the output of a context management strategy.
type StrategyResult struct {
	// Messages is the list of messages to send to the LLM.
	// Summary role messages are converted to User role for LLM compatibility.
	Messages []message.Message
	// SessionUpdate contains messages to add to the session storage.
	// This is nil for strategies that don't generate new content (truncate, sliding).
	SessionUpdate *SessionUpdate
}

// SessionUpdate contains messages to persist to the session.
type SessionUpdate struct {
	// AddMessages appends messages to the session.
	// The full conversation history is preserved for auditing.
	AddMessages []message.Message
}

// StrategyInput contains all data needed for context management.
type StrategyInput struct {
	// Messages is the list of messages to potentially trim.
	Messages []message.Message
	// SystemPrompt is the system prompt (counted but not modified).
	SystemPrompt string
	// Tools is the list of tools (counted but not modified).
	Tools []tool.BaseTool
	// Counter is the token counter to use.
	Counter TokenCounter
	// MaxTokens is the maximum allowed tokens (model context window minus reserved output).
	MaxTokens int64
}
