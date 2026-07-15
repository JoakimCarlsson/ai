package summarize

import (
	"context"
	"fmt"
	"strings"

	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
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

func (s *summarizeStrategy) Fit(
	ctx context.Context,
	input tokens.StrategyInput,
) (*tokens.StrategyResult, error) {
	// 1. Identify active context (System messages + messages since last summary)
	activeMessages := make([]message.Message, 0, len(input.Messages))
	lastSummaryIdx := -1
	for i := len(input.Messages) - 1; i >= 0; i-- {
		if input.Messages[i].Role == message.Summary {
			lastSummaryIdx = i
			break
		}
	}

	for i, msg := range input.Messages {
		switch {
		case msg.Role == message.System:
			activeMessages = append(activeMessages, msg)
		case lastSummaryIdx != -1:
			if i >= lastSummaryIdx {
				activeMessages = append(activeMessages, msg)
			}
		case msg.Role != message.Summary:
			activeMessages = append(activeMessages, msg)
		}
	}

	// 2. Check if active context fits
	count, err := input.Counter.CountTokens(ctx, tokens.CountOptions{
		Messages:     activeMessages,
		SystemPrompt: input.SystemPrompt,
		Tools:        input.Tools,
	})
	if err != nil {
		return nil, err
	}

	if count.TotalTokens <= input.MaxTokens {
		return &tokens.StrategyResult{
			Messages:      convertSummaryToUser(activeMessages),
			SessionUpdate: nil,
		}, nil
	}

	// 3. Needs summary. Identify what to summarize within the active context.
	var systemMsgs []message.Message
	var lastSummary *message.Message
	var convMsgs []message.Message

	for i := range activeMessages {
		msg := &activeMessages[i]
		switch msg.Role {
		case message.System:
			systemMsgs = append(systemMsgs, *msg)
		case message.Summary:
			lastSummary = msg
		default:
			convMsgs = append(convMsgs, *msg)
		}
	}

	splitPoint := len(convMsgs) - s.config.KeepRecent
	if splitPoint <= 0 {
		// Cannot summarize further without violating KeepRecent
		return &tokens.StrategyResult{
			Messages:      convertSummaryToUser(activeMessages),
			SessionUpdate: nil,
		}, nil
	}

	toSummarize := make([]message.Message, 0, splitPoint+1)
	if lastSummary != nil {
		toSummarize = append(toSummarize, *lastSummary)
	}
	toSummarize = append(toSummarize, convMsgs[:splitPoint]...)
	toKeep := convMsgs[splitPoint:]

	summary, err := s.generateSummary(ctx, toSummarize)
	if err != nil {
		// Fallback: return what we have if summary fails
		return &tokens.StrategyResult{
			Messages:      convertSummaryToUser(activeMessages),
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

	sessionUpdateMsgs := make([]message.Message, 0, len(toKeep)+1)
	sessionUpdateMsgs = append(sessionUpdateMsgs, summaryMsgForSession)
	sessionUpdateMsgs = append(sessionUpdateMsgs, toKeep...)

	return &tokens.StrategyResult{
		Messages: llmMessages,
		SessionUpdate: &tokens.SessionUpdate{
			PopCount:    len(toKeep),
			AddMessages: sessionUpdateMsgs,
		},
	}, nil
}

func (s *summarizeStrategy) generateSummary(
	ctx context.Context,
	msgs []message.Message,
) (string, error) {
	var sb strings.Builder
	for _, msg := range msgs {
		fmt.Fprintf(&sb, "[%s]: ", msg.Role)
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case message.TextContent:
				sb.WriteString(p.Text)
			case message.ToolCall:
				fmt.Fprintf(&sb, "[Tool call: %s]", p.Name)
			case message.ToolResult:
				fmt.Fprintf(&sb, "[Tool result: %s]", p.Name)
			case message.ReasoningContent:
				// ReasoningContent is intentionally skipped to save tokens.
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
