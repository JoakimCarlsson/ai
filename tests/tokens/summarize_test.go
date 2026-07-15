package tokens

import (
	"context"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tokens/summarize"
	"github.com/joakimcarlsson/ai/tool"
)

type mockSummarizerLLM struct {
	lastMsgs []message.Message
}

func (m *mockSummarizerLLM) SendMessages(
	_ context.Context,
	msgs []message.Message,
	_ []tool.BaseTool,
) (*llm.Response, error) {
	m.lastMsgs = msgs
	return &llm.Response{
		Content: "Mock summary",
	}, nil
}

func (m *mockSummarizerLLM) SendMessagesWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *mockSummarizerLLM) StreamResponse(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) <-chan llm.Event {
	return nil
}

func (m *mockSummarizerLLM) StreamResponseWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) <-chan llm.Event {
	return nil
}

func (m *mockSummarizerLLM) Model() model.Model {
	return model.Model{ID: "mock-summarizer"}
}

func (m *mockSummarizerLLM) SupportsStructuredOutput() bool {
	return false
}

func TestSummarizeStrategy_SkipsReasoningContent(t *testing.T) {
	mockLLM := &mockSummarizerLLM{}

	// KeepRecent is 1. We will provide 2 messages so that summarization is triggered.
	strategy := summarize.Strategy(mockLLM, summarize.KeepRecent(1))

	counter, err := tokens.NewCounter()
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	// Message with reasoning content and text content.
	msg := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ReasoningContent{Text: "SECRET_THOUGHTS_DO_NOT_LEAK"},
			message.TextContent{Text: "VISIBLE_RESPONSE_TO_USER"},
		},
	}

	input := tokens.StrategyInput{
		Messages: []message.Message{
			msg,
			message.NewUserMessage("User trigger"),
		},
		SystemPrompt: "System prompt",
		MaxTokens:    10, // low max tokens to force Fit to summarize
		Counter:      counter,
	}

	_, err = strategy.Fit(context.Background(), input)
	if err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	if len(mockLLM.lastMsgs) < 2 {
		t.Fatalf(
			"expected summarizer LLM to receive system prompt and conversation messages, got %d messages",
			len(mockLLM.lastMsgs),
		)
	}

	summaryPrompt := mockLLM.lastMsgs[1].Content().Text

	if strings.Contains(summaryPrompt, "SECRET_THOUGHTS_DO_NOT_LEAK") {
		t.Errorf(
			"expected summary prompt to skip reasoning content, but it was found: %q",
			summaryPrompt,
		)
	}

	if !strings.Contains(summaryPrompt, "VISIBLE_RESPONSE_TO_USER") {
		t.Errorf(
			"expected summary prompt to contain text content, but it was not found: %q",
			summaryPrompt,
		)
	}
}
