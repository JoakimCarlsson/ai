package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/tokens/summarize"
)

func TestAgent_SummarizeStrategy(t *testing.T) {
	ctx := context.Background()

	// 1. Setup separate mock LLMs for the summarizer and the agent.
	summarizerLLM := newMockLLM(
		mockResponse{
			Content:      "This is a summary of the first turn.",
			FinishReason: message.FinishReasonEndTurn,
		},
	)
	mockAgentLLM := newMockLLM(
		// Response for turn 1
		mockResponse{
			Content:      "Response 1",
			FinishReason: message.FinishReasonEndTurn,
		},
		// Response for turn 2 (which will include the summary)
		mockResponse{
			Content:      "I've received the summary and your new message.",
			FinishReason: message.FinishReasonEndTurn,
		},
	)

	store := session.MemoryStore()

	// Create a strategy with a low "KeepRecent"
	strat := summarize.Strategy(summarizerLLM, summarize.KeepRecent(1))

	a := agent.New(mockAgentLLM,
		agent.WithSession("test-session", store),
		agent.WithSystemPrompt("You are a test assistant."),
		// Force summary by setting a very low limit.
		// Token count for "Message 1" + system prompt + overhead will easily exceed 20.
		agent.WithContextStrategy(strat, 20),
	)

	// 2. First turn.
	// Fit will see: [System, Message 1]. convMsgs = [Message 1].
	// splitPoint = 1 - 1 = 0. No summary triggered.
	_, err := a.Chat(ctx, "Message 1")
	if err != nil {
		t.Fatalf("Turn 1 failed: %v", err)
	}

	// 3. Second turn.
	// Fit will see: [System, Message 1, Response 1, New Message].
	// convMsgs = [Message 1, Response 1, New Message].
	// splitPoint = 3 - 1 = 2.
	// Summarizes: [Message 1, Response 1].
	resp, err := a.Chat(ctx, "New Message")
	if err != nil {
		t.Fatalf("Turn 2 failed: %v", err)
	}

	if !strings.Contains(resp.Content, "I've received the summary") {
		t.Errorf(
			"Expected response acknowledging summary, got %q",
			resp.Content,
		)
	}

	// 4. Verify session state.
	sess, _ := store.Load(ctx, "test-session")
	sessMsgs, err := sess.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get session messages: %v", err)
	}

	// Expected session messages:
	// 1. User: Message 1
	// 2. Assistant: Response 1
	// 3. Summary: This is a summary...
	// 4. User: New Message
	// 5. Assistant: I've received the summary...
	if len(sessMsgs) != 5 {
		t.Errorf("Expected 5 messages in session, got %d", len(sessMsgs))
	}

	hasSummary := false
	for _, m := range sessMsgs {
		if m.Role == message.Summary {
			hasSummary = true
			if !strings.Contains(m.Content().Text, "This is a summary") {
				t.Errorf("Unexpected summary content: %q", m.Content().Text)
			}
		}
	}

	if !hasSummary {
		t.Error("Session store does not contain a summary message")
	}

	// 5. Verify LLM calls.

	// Agent should have 2 chat calls.
	if mockAgentLLM.CallCount() != 2 {
		t.Errorf("Expected 2 Agent LLM calls, got %d", mockAgentLLM.CallCount())
	}

	// Summarizer should have 1 call.
	if summarizerLLM.CallCount() != 1 {
		t.Errorf(
			"Expected 1 Summarizer LLM call, got %d",
			summarizerLLM.CallCount(),
		)
	}

	// Verify turn 2 LLM prompt received the summary.
	agentTurn2Msgs := mockAgentLLM.calls[1]
	foundSummaryInPrompt := false
	foundOldMsgInPrompt := false
	for _, m := range agentTurn2Msgs {
		if strings.Contains(m.Content().Text, "This is a summary") {
			foundSummaryInPrompt = true
		}
		if strings.Contains(m.Content().Text, "Message 1") {
			foundOldMsgInPrompt = true
		}
	}

	if !foundSummaryInPrompt {
		t.Error("Summary was not sent to the Agent LLM in Turn 2")
	}
	if foundOldMsgInPrompt {
		t.Error(
			"Old message 1 was sent to the Agent LLM in Turn 2 despite being summarized",
		)
	}
}
