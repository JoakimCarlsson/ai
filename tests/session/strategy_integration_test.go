package session

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/tokens/summarize"
)

// runSummarizeStrategyTest drives a real agent.Agent backed by the given store,
// chatting until the summarize strategy compacts the history, then verifies the
// store persisted the compaction correctly. Running it against both FileStore and
// MemoryStore exercises summary-message round-tripping through each backend via
// the actual agent persistence path (PopCount + AddMessages), rather than a
// hand-simulated session update.
func runSummarizeStrategyTest(t *testing.T, store session.Store) {
	ctx := context.Background()

	const sessionID = "strategy-test"
	summarizer := &bugMockSummarizer{}
	// Keep only the most recent message so a summary triggers quickly.
	strat := summarize.Strategy(summarizer, summarize.KeepRecent(1))

	a := agent.New(
		&mockAgentLLM{t: t},
		agent.WithSession(sessionID, store),
		agent.WithContextStrategy(strat, 100),
	)

	// Chat until the active context exceeds the limit and a summary is produced.
	turn := 1
	for summarizer.callCount == 0 && turn <= 10 {
		msg := fmt.Sprintf(
			"Hello, this is message %d. Please remember it.",
			turn,
		)
		if _, err := a.Chat(ctx, msg); err != nil {
			t.Fatalf("Chat turn %d failed: %v", turn, err)
		}
		turn++
	}
	if summarizer.callCount == 0 {
		t.Fatal("expected summarizer to be triggered within 10 turns")
	}

	// Reload from the store to confirm the compaction round-trips through the backend.
	sess, err := store.Load(ctx, sessionID)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	saved, err := sess.GetMessages(ctx, nil)
	if err != nil {
		t.Fatalf("failed to get saved messages: %v", err)
	}

	// Exactly one summary message should be persisted, with the expected content.
	summaryCount := 0
	summaryIdx := -1
	for i, m := range saved {
		if m.Role == message.Summary {
			summaryCount++
			summaryIdx = i
		}
	}
	if summaryCount != 1 {
		t.Fatalf(
			"expected exactly 1 persisted summary message, got %d",
			summaryCount,
		)
	}
	if !strings.Contains(
		saved[summaryIdx].Content().Text,
		"This is the summary content",
	) {
		t.Errorf(
			"unexpected summary content: %q",
			saved[summaryIdx].Content().Text,
		)
	}

	// A fresh agent on the same store+id reloads the session from the backend
	// (for FileStore this deserializes from disk). The persisted Summary acts as
	// an anchor: a short follow-up must fit and must NOT re-trigger summarization.
	// If the backend dropped the Summary role on round-trip, the anchor would be
	// lost and the strategy would summarize again, failing this assertion.
	summarizer.callCount = 0
	reloaded := agent.New(
		&mockAgentLLM{t: t},
		agent.WithSession(sessionID, store),
		agent.WithContextStrategy(strat, 100),
	)
	if _, err := reloaded.Chat(ctx, "Short follow-up"); err != nil {
		t.Fatalf("follow-up chat failed: %v", err)
	}
	if summarizer.callCount != 0 {
		t.Errorf(
			"summarizer re-triggered after reload on a turn that fits (called %d times)",
			summarizer.callCount,
		)
	}
}

func TestFileSession_SummarizeStrategy(t *testing.T) {
	dir := t.TempDir()
	store := session.FileStore(dir)
	if store == nil {
		t.Fatal("failed to create FileStore")
	}
	runSummarizeStrategyTest(t, store)
}

func TestMemorySession_SummarizeStrategy(t *testing.T) {
	store := session.MemoryStore()
	runSummarizeStrategyTest(t, store)
}
