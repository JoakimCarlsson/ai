package tokens

import (
	"context"
	"encoding/json"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

const (
	SystemMessageOverhead  int64 = 15
	MessageOverhead        int64 = 10
	ToolCallOverhead       int64 = 10
	ToolResultOverhead     int64 = 10
	ToolDefinitionOverhead int64 = 20
)

// TokenCounter provides methods for counting tokens in conversations.
type TokenCounter interface {
	CountTokens(ctx context.Context, opts CountOptions) (*TokenCount, error)
}

// CountOptions contains the inputs for token counting.
type CountOptions struct {
	Messages     []message.Message
	SystemPrompt string
	Tools        []tool.BaseTool
}

// TokenCount contains the breakdown of token counts.
type TokenCount struct {
	// SystemTokens is the token count for the system prompt.
	SystemTokens int64
	// MessageTokens is the token count for all messages.
	MessageTokens int64
	// ToolTokens is the token count for tool definitions.
	ToolTokens int64
	// TotalTokens is the sum of all token counts.
	TotalTokens int64
}

// Counter implements TokenCounter using the BPE tokenizer.
type Counter struct {
	tokenizer *BPETokenizer
}

// NewCounter creates a new token counter.
func NewCounter() (*Counter, error) {
	tokenizer, err := NewBPETokenizer()
	if err != nil {
		return nil, err
	}
	return &Counter{tokenizer: tokenizer}, nil
}

// CountTokens counts tokens for messages, system prompt, and tools.
func (c *Counter) CountTokens(ctx context.Context, opts CountOptions) (*TokenCount, error) {
	var result TokenCount

	if opts.SystemPrompt != "" {
		result.SystemTokens = int64(c.tokenizer.Count(opts.SystemPrompt)) + SystemMessageOverhead
	}

	for _, msg := range opts.Messages {
		if msg.Role == message.System {
			continue
		}

		result.MessageTokens += MessageOverhead

		for _, part := range msg.Parts {
			switch p := part.(type) {
			case message.TextContent:
				result.MessageTokens += int64(c.tokenizer.Count(p.Text))
			case message.BinaryContent:
				result.MessageTokens += EstimateImageTokens(p)
			case message.ImageURLContent:
				result.MessageTokens += DefaultImageTokens
			case message.ToolCall:
				result.MessageTokens += int64(c.tokenizer.Count(p.Name))
				result.MessageTokens += int64(c.tokenizer.Count(p.Input))
				result.MessageTokens += ToolCallOverhead
			case message.ToolResult:
				result.MessageTokens += int64(c.tokenizer.Count(p.Content))
				result.MessageTokens += ToolResultOverhead
			}
		}
	}

	for _, t := range opts.Tools {
		info := t.Info()
		result.ToolTokens += int64(c.tokenizer.Count(info.Name))
		result.ToolTokens += int64(c.tokenizer.Count(info.Description))
		result.ToolTokens += c.countParameterTokens(info.Parameters)
		result.ToolTokens += ToolDefinitionOverhead
	}

	result.TotalTokens = result.SystemTokens + result.MessageTokens + result.ToolTokens
	return &result, nil
}

func (c *Counter) countParameterTokens(params map[string]any) int64 {
	if params == nil {
		return 0
	}

	var tokens int64

	properties, ok := params["properties"].(map[string]any)
	if !ok {
		if data, err := json.Marshal(params); err == nil {
			tokens += int64(c.tokenizer.Count(string(data)))
		}
		return tokens
	}

	for propName, propSchema := range properties {
		tokens += int64(c.tokenizer.Count(propName))

		schema, ok := propSchema.(map[string]any)
		if !ok {
			continue
		}

		if t, ok := schema["type"].(string); ok {
			tokens += int64(c.tokenizer.Count(t))
		}

		if desc, ok := schema["description"].(string); ok {
			tokens += int64(c.tokenizer.Count(desc))
		}

		if enum, ok := schema["enum"].([]any); ok {
			for _, v := range enum {
				if s, ok := v.(string); ok {
					tokens += int64(c.tokenizer.Count(s))
				} else {
					tokens += 2
				}
			}
		}

		if nested, ok := schema["properties"].(map[string]any); ok {
			tokens += c.countParameterTokens(map[string]any{"properties": nested})
		}

		if items, ok := schema["items"].(map[string]any); ok {
			tokens += c.countParameterTokens(items)
		}
	}

	return tokens
}
