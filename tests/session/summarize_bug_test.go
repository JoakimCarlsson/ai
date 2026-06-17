package session

import (
	"context"
	"fmt"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tokens/summarize"
	"github.com/joakimcarlsson/ai/tool"
)

type bugMockSummarizer struct {
	callCount int
	lastMsgs  []message.Message
}

func (m *bugMockSummarizer) SendMessages(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	m.callCount++
	m.lastMsgs = msgs
	return &llm.Response{
		Content: "This is the summary content. It replaces the previous messages to save space.",
	}, nil
}

func (m *bugMockSummarizer) SendMessagesWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *bugMockSummarizer) StreamResponse(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	return nil
}

func (m *bugMockSummarizer) StreamResponseWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	return nil
}

func (m *bugMockSummarizer) Model() model.Model {
	return model.Model{ID: "mock-summarizer"}
}

func (m *bugMockSummarizer) SupportsStructuredOutput() bool {
	return false
}

// mockAgentLLM acts as the primary LLM for the agent
type mockAgentLLM struct {
	t *testing.T
}

func (m *mockAgentLLM) SendMessages(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	m.t.Logf("--> mockAgentLLM called with %d messages", len(msgs))
	return &llm.Response{
		Content: "Agent Response",
	}, nil
}

func (m *mockAgentLLM) SendMessagesWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *mockAgentLLM) StreamResponse(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	return nil
}

func (m *mockAgentLLM) StreamResponseWithStructuredOutput(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	return nil
}

func (m *mockAgentLLM) Model() model.Model {
	return model.Model{ID: "mock-agent-llm"}
}

func (m *mockAgentLLM) SupportsStructuredOutput() bool {
	return false
}

func TestSummarizeStrategy_ReproductionBug(t *testing.T) {
	ctx := context.Background()
	store := session.MemoryStore()

	summarizer := &bugMockSummarizer{}
	// Keep 1 most recent message pair.
	strat := summarize.Strategy(summarizer, summarize.KeepRecent(1))

	a := agent.New(
		&mockAgentLLM{t: t},
		agent.WithSession("bug-test2", store),
		agent.WithContextStrategy(strat, 100),
	)

	// Chat until we trigger a summary
	turn := 1
	for summarizer.callCount == 0 && turn <= 10 {
		msg := fmt.Sprintf(
			"Hello, this is message %d. Please remember it.",
			turn,
		)
		_, err := a.Chat(ctx, msg)
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}
		turn++
	}

	if summarizer.callCount == 0 {
		t.Fatalf(
			"Expected summary to be triggered, but it was not after 10 turns.",
		)
	}

	sess, _ := store.Load(ctx, "bug-test2")
	rawMsgs, _ := sess.GetMessages(ctx, nil)
	t.Logf("Session State before Next Turn: %d messages", len(rawMsgs))
	for i, m := range rawMsgs {
		t.Logf(" - Session[%d] %s: %s", i, m.Role, m.Content().Text)
	}

	// Critical check for continuous re-summarization
	// Active context = [Summary, Kept Assistant, New Msg].
	// Should fit well within 100 tokens.
	t.Logf("--- Turn %d (Checking Compaction) ---", turn)
	summarizer.callCount = 0

	peekMsgs, _ := a.PeekContextMessages(ctx, "Short M4")
	t.Logf("Peek: %d messages evaluated", len(peekMsgs))
	for i, m := range peekMsgs {
		t.Logf(" - [%d] %s: %s", i, m.Role, m.Content().Text)
	}

	c, _ := tokens.NewCounter()
	count, _ := c.CountTokens(ctx, tokens.CountOptions{Messages: peekMsgs})
	t.Logf("Peek actual tokens: %d", count.TotalTokens)

	_, err := a.Chat(ctx, "Short M4")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if summarizer.callCount != 0 {
		t.Errorf(
			"Bug 1 Confirmed: Summary triggered again even though active context should fit. Summarizer called %d times.",
			summarizer.callCount,
		)
	}
}
