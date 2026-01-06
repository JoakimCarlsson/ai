package sliding

import (
	"context"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tokens"
)

type slidingStrategy struct {
	config *Config
}

// Strategy returns a sliding window strategy that keeps the last N messages.
func Strategy(opts ...Option) tokens.Strategy {
	return &slidingStrategy{config: Apply(opts...)}
}

func (s *slidingStrategy) Fit(ctx context.Context, input tokens.StrategyInput) (*tokens.StrategyResult, error) {
	var systemMsgs, convMsgs []message.Message

	for _, msg := range input.Messages {
		if msg.Role == message.System {
			systemMsgs = append(systemMsgs, msg)
		} else {
			convMsgs = append(convMsgs, msg)
		}
	}

	if len(convMsgs) > s.config.KeepLast {
		convMsgs = convMsgs[len(convMsgs)-s.config.KeepLast:]
	}

	return &tokens.StrategyResult{
		Messages:      append(systemMsgs, convMsgs...),
		SessionUpdate: nil,
	}, nil
}
