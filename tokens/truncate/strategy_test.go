package truncate

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tokens"
)

type mockCounter struct {
	tokensPerMessage int64
}

func (c *mockCounter) CountTokens(
	_ context.Context,
	opts tokens.CountOptions,
) (*tokens.TokenCount, error) {
	msgTokens := int64(len(opts.Messages)) * c.tokensPerMessage
	return &tokens.TokenCount{
		MessageTokens: msgTokens,
		TotalTokens:   msgTokens,
	}, nil
}

func makeMessages(roles ...message.Role) []message.Message {
	msgs := make([]message.Message, len(roles))
	for i, r := range roles {
		switch r {
		case message.System:
			msgs[i] = message.NewSystemMessage("system prompt")
		case message.User:
			msgs[i] = message.NewUserMessage("user message")
		case message.Assistant:
			msgs[i] = message.NewAssistantMessage()
			msgs[i].AppendContent("assistant reply")
		default:
			msgs[i] = message.Message{Role: r}
		}
	}
	return msgs
}

func TestTruncate_UnderBudget(t *testing.T) {
	s := Strategy()
	counter := &mockCounter{tokensPerMessage: 10}
	msgs := makeMessages(message.System, message.User, message.Assistant)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != 3 {
		t.Errorf(
			"expected all 3 messages retained, got %d",
			len(result.Messages),
		)
	}
}

func TestTruncate_RemovesOldest(t *testing.T) {
	s := Strategy()
	counter := &mockCounter{tokensPerMessage: 100}
	msgs := makeMessages(
		message.System,
		message.User, message.Assistant,
		message.User, message.Assistant,
		message.User, message.Assistant,
	)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 500,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) >= 7 {
		t.Errorf(
			"expected fewer messages after truncation, got %d",
			len(result.Messages),
		)
	}
	if result.Messages[0].Role != message.System {
		t.Error("expected system message to remain first")
	}
}

func TestTruncate_PreservesSystemMessages(t *testing.T) {
	s := Strategy()
	counter := &mockCounter{tokensPerMessage: 100}
	msgs := makeMessages(
		message.System,
		message.User, message.Assistant,
		message.User, message.Assistant,
	)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 200,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, msg := range result.Messages {
		if msg.Role == message.System {
			return
		}
	}
	t.Error("system message was removed during truncation")
}

func TestTruncate_PreservePairs(t *testing.T) {
	s := Strategy(PreservePairs())
	counter := &mockCounter{tokensPerMessage: 100}
	msgs := makeMessages(
		message.System,
		message.User, message.Assistant,
		message.User, message.Assistant,
		message.User, message.Assistant,
	)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 500,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nonSystem := 0
	for _, msg := range result.Messages {
		if msg.Role != message.System {
			nonSystem++
		}
	}
	if nonSystem%2 != 0 {
		t.Errorf(
			"expected even number of non-system messages (pairs), got %d",
			nonSystem,
		)
	}
}

func TestTruncate_MinMessages(t *testing.T) {
	s := Strategy(MinMessages(3))
	counter := &mockCounter{tokensPerMessage: 1000}
	msgs := makeMessages(
		message.System,
		message.User, message.Assistant,
		message.User, message.Assistant,
	)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) < 3 {
		t.Errorf(
			"expected at least 3 messages with MinMessages(3), got %d",
			len(result.Messages),
		)
	}
}

func TestTruncate_NilSessionUpdate(t *testing.T) {
	s := Strategy()
	counter := &mockCounter{tokensPerMessage: 100}
	msgs := makeMessages(message.System, message.User, message.Assistant)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SessionUpdate != nil {
		t.Error("expected nil SessionUpdate for truncate strategy")
	}
}
