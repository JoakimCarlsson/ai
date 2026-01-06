package truncate

import (
	"context"
	"slices"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tokens"
)

type truncateStrategy struct {
	config *Config
}

// Strategy returns a truncate strategy that removes oldest messages.
func Strategy(opts ...Option) tokens.Strategy {
	return &truncateStrategy{config: Apply(opts...)}
}

func (s *truncateStrategy) Fit(ctx context.Context, input tokens.StrategyInput) (*tokens.StrategyResult, error) {
	result := slices.Clone(input.Messages)

	for len(result) > s.config.MinMessages {
		count, err := input.Counter.CountTokens(ctx, tokens.CountOptions{
			Messages:     result,
			SystemPrompt: input.SystemPrompt,
			Tools:        input.Tools,
		})
		if err != nil {
			return nil, err
		}

		if count.TotalTokens <= input.MaxTokens {
			break
		}

		result = s.removeOldest(result)
	}

	return &tokens.StrategyResult{
		Messages:      result,
		SessionUpdate: nil,
	}, nil
}

func (s *truncateStrategy) removeOldest(msgs []message.Message) []message.Message {
	if len(msgs) == 0 {
		return msgs
	}

	startIdx := 0
	for startIdx < len(msgs) && msgs[startIdx].Role == message.System {
		startIdx++
	}

	if startIdx >= len(msgs) {
		return msgs
	}

	if !s.config.PreservePairs {
		return append(msgs[:startIdx], msgs[startIdx+1:]...)
	}

	first := msgs[startIdx]

	if first.Role == message.User && startIdx+1 < len(msgs) &&
		msgs[startIdx+1].Role == message.Assistant {
		return append(msgs[:startIdx], msgs[startIdx+2:]...)
	}

	if first.Role == message.Assistant && len(first.ToolCalls()) > 0 {
		endIdx := startIdx + 1
		for endIdx < len(msgs) && msgs[endIdx].Role == message.Tool {
			endIdx++
		}
		return append(msgs[:startIdx], msgs[endIdx:]...)
	}

	return append(msgs[:startIdx], msgs[startIdx+1:]...)
}
