package sliding

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tokens"
)

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

func TestSliding_KeepLast(t *testing.T) {
	s := Strategy(KeepLast(3))
	msgs := makeMessages(
		message.System,
		message.User, message.Assistant,
		message.User, message.Assistant,
		message.User, message.Assistant,
		message.User, message.Assistant,
		message.User, message.Assistant,
	)

	result, err := s.Fit(
		context.Background(),
		tokens.StrategyInput{Messages: msgs},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nonSystem := 0
	for _, msg := range result.Messages {
		if msg.Role != message.System {
			nonSystem++
		}
	}
	if nonSystem != 3 {
		t.Errorf("expected 3 non-system messages, got %d", nonSystem)
	}
	if result.Messages[0].Role != message.System {
		t.Error("expected system message first")
	}
}

func TestSliding_SystemPreserved(t *testing.T) {
	s := Strategy(KeepLast(2))
	msgs := makeMessages(
		message.System,
		message.User, message.Assistant,
		message.User, message.Assistant,
		message.User, message.Assistant,
	)

	result, err := s.Fit(
		context.Background(),
		tokens.StrategyInput{Messages: msgs},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Messages) != 3 {
		t.Fatalf(
			"expected 3 messages (1 system + 2 kept), got %d",
			len(result.Messages),
		)
	}
	if result.Messages[0].Role != message.System {
		t.Error("expected system message to be preserved at position 0")
	}
}

func TestSliding_UnderLimit(t *testing.T) {
	s := Strategy(KeepLast(20))
	msgs := makeMessages(message.System, message.User, message.Assistant)

	result, err := s.Fit(
		context.Background(),
		tokens.StrategyInput{Messages: msgs},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Messages) != 3 {
		t.Errorf(
			"expected all 3 messages retained when under limit, got %d",
			len(result.Messages),
		)
	}
}

func TestSliding_Default(t *testing.T) {
	s := Strategy()
	roles := []message.Role{message.System}
	for range 15 {
		roles = append(roles, message.User, message.Assistant)
	}
	msgs := makeMessages(roles...)

	result, err := s.Fit(
		context.Background(),
		tokens.StrategyInput{Messages: msgs},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nonSystem := 0
	for _, msg := range result.Messages {
		if msg.Role != message.System {
			nonSystem++
		}
	}
	if nonSystem != 10 {
		t.Errorf(
			"expected 10 non-system messages (default KeepLast=10), got %d",
			nonSystem,
		)
	}
}

func TestSliding_NilSessionUpdate(t *testing.T) {
	s := Strategy(KeepLast(3))
	msgs := makeMessages(
		message.System,
		message.User,
		message.Assistant,
		message.User,
		message.Assistant,
	)

	result, err := s.Fit(
		context.Background(),
		tokens.StrategyInput{Messages: msgs},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SessionUpdate != nil {
		t.Error("expected nil SessionUpdate for sliding strategy")
	}
}
