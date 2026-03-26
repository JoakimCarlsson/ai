package summarize

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
)

type sumMockLLM struct {
	content string
	err     error
}

func (m *sumMockLLM) SendMessages(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) (*llm.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.Response{Content: m.content}, nil
}

func (m *sumMockLLM) SendMessagesWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *sumMockLLM) StreamResponse(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *sumMockLLM) StreamResponseWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *sumMockLLM) Model() model.Model {
	return model.Model{ID: "mock", Provider: "mock"}
}

func (m *sumMockLLM) SupportsStructuredOutput() bool { return false }

type mockCounter struct {
	tokensPerMessage int64
}

func (c *mockCounter) CountTokens(
	_ context.Context,
	opts tokens.CountOptions,
) (*tokens.TokenCount, error) {
	total := int64(len(opts.Messages)) * c.tokensPerMessage
	return &tokens.TokenCount{MessageTokens: total, TotalTokens: total}, nil
}

func makeConversation(n int) []message.Message {
	msgs := []message.Message{message.NewSystemMessage("system")}
	for i := range n {
		msgs = append(
			msgs,
			message.NewUserMessage(fmt.Sprintf("user msg %d", i)),
		)
		a := message.NewAssistantMessage()
		a.AppendContent(fmt.Sprintf("assistant msg %d", i))
		msgs = append(msgs, a)
	}
	return msgs
}

func TestSummarize_UnderBudget(t *testing.T) {
	mock := &sumMockLLM{content: "should not be called"}
	s := Strategy(mock)
	counter := &mockCounter{tokensPerMessage: 10}
	msgs := makeConversation(3)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 10000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != len(msgs) {
		t.Errorf(
			"expected all %d messages retained, got %d",
			len(msgs),
			len(result.Messages),
		)
	}
	if result.SessionUpdate != nil {
		t.Error("expected nil SessionUpdate when under budget")
	}
}

func TestSummarize_OverBudget(t *testing.T) {
	mock := &sumMockLLM{content: "The user discussed topics A and B."}
	s := Strategy(mock, KeepRecent(2))
	counter := &mockCounter{tokensPerMessage: 100}
	msgs := makeConversation(10)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SessionUpdate == nil {
		t.Fatal("expected non-nil SessionUpdate when summarizing")
	}
	if len(result.SessionUpdate.AddMessages) != 1 {
		t.Fatalf(
			"expected 1 session message, got %d",
			len(result.SessionUpdate.AddMessages),
		)
	}
	if result.SessionUpdate.AddMessages[0].Role != message.Summary {
		t.Errorf(
			"expected summary role in session update, got %s",
			result.SessionUpdate.AddMessages[0].Role,
		)
	}

	var hasSummaryUser bool
	for _, msg := range result.Messages {
		if msg.Role == message.User {
			txt := msg.Content().Text
			if strings.Contains(txt, "Previous conversation summary:") {
				hasSummaryUser = true
			}
		}
	}
	if !hasSummaryUser {
		t.Error("expected injected summary as user message in output")
	}

	nonSystem := 0
	for _, msg := range result.Messages {
		if msg.Role != message.System &&
			!strings.Contains(
				msg.Content().Text,
				"Previous conversation summary:",
			) {
			nonSystem++
		}
	}
	if nonSystem != 2 {
		t.Errorf("expected 2 recent messages kept, got %d", nonSystem)
	}
}

func TestSummarize_SummaryToUserConversion(t *testing.T) {
	mock := &sumMockLLM{content: "unused"}
	s := Strategy(mock)
	counter := &mockCounter{tokensPerMessage: 10}

	msgs := []message.Message{
		message.NewSystemMessage("system"),
		message.NewSummaryMessage(
			"Previous conversation summary:\nSomething happened",
		),
		message.NewUserMessage("hello"),
	}

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 10000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, msg := range result.Messages {
		if msg.Role == message.Summary {
			t.Error("summary role should have been converted to user")
		}
	}
	if result.Messages[1].Role != message.User {
		t.Errorf(
			"expected summary converted to user at index 1, got %s",
			result.Messages[1].Role,
		)
	}
}

func TestSummarize_KeepRecentOption(t *testing.T) {
	mock := &sumMockLLM{content: "compressed history"}
	s := Strategy(mock, KeepRecent(2))
	counter := &mockCounter{tokensPerMessage: 100}
	msgs := makeConversation(8)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recentCount := 0
	for _, msg := range result.Messages {
		if msg.Role != message.System &&
			!strings.Contains(
				msg.Content().Text,
				"Previous conversation summary:",
			) {
			recentCount++
		}
	}
	if recentCount != 2 {
		t.Errorf(
			"expected 2 recent messages with KeepRecent(2), got %d",
			recentCount,
		)
	}
}

func TestSummarize_LLMErrorFallback(t *testing.T) {
	mock := &sumMockLLM{err: fmt.Errorf("LLM unavailable")}
	s := Strategy(mock, KeepRecent(2))
	counter := &mockCounter{tokensPerMessage: 100}
	msgs := makeConversation(8)

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Messages) != len(msgs) {
		t.Errorf(
			"expected original messages returned on LLM error, got %d vs %d",
			len(result.Messages),
			len(msgs),
		)
	}
	if result.SessionUpdate != nil {
		t.Error("expected nil SessionUpdate on LLM error fallback")
	}
}

func TestSummarize_PriorSummaryFolded(t *testing.T) {
	var capturedInput string
	mock := &sumMockLLM{content: "merged summary"}
	origSend := mock.SendMessages
	_ = origSend

	capturingLLM := &capturingSumLLM{
		content: "merged summary",
		onCall: func(msgs []message.Message) {
			for _, m := range msgs {
				if m.Role == message.User {
					capturedInput = m.Content().Text
				}
			}
		},
	}

	s := Strategy(capturingLLM, KeepRecent(2))
	counter := &mockCounter{tokensPerMessage: 100}

	msgs := []message.Message{
		message.NewSystemMessage("system"),
		message.NewSummaryMessage("Old summary content"),
		message.NewUserMessage("msg 1"),
		message.NewUserMessage("msg 2"),
		message.NewUserMessage("msg 3"),
		message.NewUserMessage("msg 4"),
	}

	result, err := s.Fit(context.Background(), tokens.StrategyInput{
		Messages:  msgs,
		Counter:   counter,
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedInput, "Old summary content") {
		t.Error("expected prior summary to be included in summarization input")
	}
	if result.SessionUpdate == nil {
		t.Fatal("expected SessionUpdate with new summary")
	}
}

type capturingSumLLM struct {
	content string
	onCall  func(msgs []message.Message)
}

func (m *capturingSumLLM) SendMessages(
	_ context.Context,
	msgs []message.Message,
	_ []tool.BaseTool,
) (*llm.Response, error) {
	if m.onCall != nil {
		m.onCall(msgs)
	}
	return &llm.Response{Content: m.content}, nil
}

func (m *capturingSumLLM) SendMessagesWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *capturingSumLLM) StreamResponse(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *capturingSumLLM) StreamResponseWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *capturingSumLLM) Model() model.Model {
	return model.Model{ID: "mock", Provider: "mock"}
}

func (m *capturingSumLLM) SupportsStructuredOutput() bool { return false }
