package tokens

import (
	"context"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// Strategy defines how to manage context when it exceeds the model's limit.
type Strategy interface {
	Fit(ctx context.Context, input StrategyInput) ([]message.Message, error)
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
