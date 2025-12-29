package summarize

import (
	"context"
	"fmt"
	"strings"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tokens"
)

const defaultSummaryPrompt = `Summarize the following conversation concisely. Include:
- Key decisions made
- Important facts mentioned
- Current context and state
- Any unresolved questions or pending items

Keep the summary focused and informative.`

type summarizeStrategy struct {
	llm    llm.LLM
	config *Config
}

// Strategy returns a summarize strategy that uses an LLM to compress older messages.
func Strategy(l llm.LLM, opts ...Option) tokens.Strategy {
	return &summarizeStrategy{
		llm:    l,
		config: Apply(opts...),
	}
}

func (s *summarizeStrategy) Fit(ctx context.Context, input tokens.StrategyInput) (*tokens.StrategyResult, error) {
	count, err := input.Counter.CountTokens(ctx, tokens.CountOptions{
		Messages:     input.Messages,
		SystemPrompt: input.SystemPrompt,
		Tools:        input.Tools,
	})
	if err != nil {
		return nil, err
	}

	if count.TotalTokens <= input.MaxTokens {
		return &tokens.StrategyResult{
			Messages:      convertSummaryToUser(input.Messages),
			SessionUpdate: nil,
		}, nil
	}

	var systemMsgs, summaryMsgs, convMsgs []message.Message
	for _, msg := range input.Messages {
		switch msg.Role {
		case message.System:
			systemMsgs = append(systemMsgs, msg)
		case message.Summary:
			summaryMsgs = append(summaryMsgs, msg)
		default:
			convMsgs = append(convMsgs, msg)
		}
	}

	splitPoint := len(convMsgs) - s.config.KeepRecent
	if splitPoint <= 0 {
		return &tokens.StrategyResult{
			Messages:      convertSummaryToUser(input.Messages),
			SessionUpdate: nil,
		}, nil
	}

	toSummarize := convMsgs[:splitPoint]
	toKeep := convMsgs[splitPoint:]

	if len(summaryMsgs) > 0 {
		toSummarize = append(summaryMsgs, toSummarize...)
	}

	summary, err := s.generateSummary(ctx, toSummarize)
	if err != nil {
		return &tokens.StrategyResult{
			Messages:      convertSummaryToUser(input.Messages),
			SessionUpdate: nil,
		}, nil
	}

	summaryContent := "Previous conversation summary:\n" + summary
	summaryMsgForSession := message.NewSummaryMessage(summaryContent)
	summaryMsgForLLM := message.NewUserMessage(summaryContent)

	llmMessages := make([]message.Message, 0, len(systemMsgs)+1+len(toKeep))
	llmMessages = append(llmMessages, systemMsgs...)
	llmMessages = append(llmMessages, summaryMsgForLLM)
	llmMessages = append(llmMessages, toKeep...)

	return &tokens.StrategyResult{
		Messages: llmMessages,
		SessionUpdate: &tokens.SessionUpdate{
			AddMessages: []message.Message{summaryMsgForSession},
		},
	}, nil
}

func (s *summarizeStrategy) generateSummary(ctx context.Context, msgs []message.Message) (string, error) {
	var sb strings.Builder
	for _, msg := range msgs {
		sb.WriteString(fmt.Sprintf("[%s]: ", msg.Role))
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case message.TextContent:
				sb.WriteString(p.Text)
			case message.ToolCall:
				sb.WriteString(fmt.Sprintf("[Tool call: %s]", p.Name))
			case message.ToolResult:
				sb.WriteString(fmt.Sprintf("[Tool result: %s]", p.Name))
			}
		}
		sb.WriteString("\n\n")
	}

	summaryMessages := []message.Message{
		message.NewSystemMessage(defaultSummaryPrompt),
		message.NewUserMessage(sb.String()),
	}

	resp, err := s.llm.SendMessages(ctx, summaryMessages, nil)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func convertSummaryToUser(msgs []message.Message) []message.Message {
	result := make([]message.Message, len(msgs))
	for i, msg := range msgs {
		if msg.Role == message.Summary {
			result[i] = message.Message{
				Role:      message.User,
				Parts:     msg.Parts,
				Model:     msg.Model,
				CreatedAt: msg.CreatedAt,
			}
		} else {
			result[i] = msg
		}
	}
	return result
}
