package voice

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/memory"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/voice"
)

// fakeMemory is a minimal memory.Store for tests. Search returns whatever
// has been seeded; Store/Update/Delete record what was attempted so tests
// can assert on the dedup / extract paths.
type fakeMemory struct {
	mu       sync.Mutex
	entries  []memory.Entry
	stored   []memory.Entry
	updated  []memory.Entry
	deleted  []string
	searches int
}

func newFakeMemory(seed ...string) *fakeMemory {
	m := &fakeMemory{}
	for i, s := range seed {
		m.entries = append(m.entries, memory.Entry{
			ID:        idForIndex(i),
			Content:   s,
			OwnerID:   "test",
			Score:     1.0,
			CreatedAt: time.Now(),
		})
	}
	return m
}

func idForIndex(i int) string {
	return "mem-" + string(rune('a'+i))
}

func (f *fakeMemory) Store(
	_ context.Context,
	ownerID, fact string,
	metadata map[string]any,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := idForIndex(len(f.entries))
	entry := memory.Entry{
		ID:        id,
		Content:   fact,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}
	f.entries = append(f.entries, entry)
	f.stored = append(f.stored, entry)
	return nil
}

func (f *fakeMemory) Search(
	_ context.Context,
	_ string,
	_ string,
	_ int,
) ([]memory.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searches++
	out := make([]memory.Entry, len(f.entries))
	copy(out, f.entries)
	return out, nil
}

func (f *fakeMemory) GetAll(
	_ context.Context,
	_ string,
	_ int,
) ([]memory.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]memory.Entry, len(f.entries))
	copy(out, f.entries)
	return out, nil
}

func (f *fakeMemory) Delete(_ context.Context, memoryID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, memoryID)
	return nil
}

func (f *fakeMemory) Update(
	_ context.Context,
	memoryID, fact string,
	metadata map[string]any,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updated = append(f.updated, memory.Entry{
		ID:       memoryID,
		Content:  fact,
		Metadata: metadata,
	})
	return nil
}

func (f *fakeMemory) searchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.searches
}

//  1. Recall: configured memories appear as a system message in the LLM
//     input. The text matches the formatted "Relevant memories about this
//     user:" block.
func TestMemory_RecallInjectsSystemMessage(t *testing.T) {
	mem := newFakeMemory(
		"the user's name is Alice",
		"Alice prefers brevity",
	)

	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithSystemPrompt("you are helpful"),
		voice.WithMemory("user-1", mem),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if mem.searchCount() < 1 {
		t.Fatalf(
			"expected memory.Search called >= 1, got %d",
			mem.searchCount(),
		)
	}

	msgs := llmFake.lastMessages()
	var sawRecall bool
	for _, m := range msgs {
		if m.Role != message.System {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok &&
				strings.Contains(
					tc.Text,
					"Relevant memories about this user",
				) &&
				strings.Contains(tc.Text, "Alice") {
				sawRecall = true
			}
		}
	}
	if !sawRecall {
		t.Fatalf("expected system message with recall context; got %+v", msgs)
	}
}

//  2. AutoExtract: after a successful user turn, the runner spawns a
//     goroutine that calls memory.ExtractFacts (which calls SendMessages
//     on the LLM). We verify by seeing SendMessages get called on the
//     fake LLM after the turn ends.
func TestMemory_AutoExtractFiresAfterTurn(t *testing.T) {
	mem := newFakeMemory()
	store := session.MemoryStore()

	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithSystemPrompt("sys"),
		voice.WithSession("sess-extract", store),
		voice.WithMemory("user-1", mem, memory.AutoExtract()),
	)
	defer a.cancel()

	a.stt.pushFinal("my favourite colour is blue")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	// SendMessages may run asynchronously; allow up to 2s.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if llmFake.sendMessageCallCount() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if llmFake.sendMessageCallCount() == 0 {
		t.Fatalf("expected SendMessages invoked by extractor; never fired")
	}

	a.cancel()
	_ = a.conv.Wait()
}

//  3. Without AutoExtract but with memoryID set, memory.Tools (store /
//     recall / replace / delete) are added to the LLM tool list so the
//     LLM can manage memory itself.
func TestMemory_ManualToolsExposedWhenNotAutoExtract(t *testing.T) {
	mem := newFakeMemory()

	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithSystemPrompt("sys"),
		voice.WithMemory("user-1", mem),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	tools := llmFake.lastToolList()
	want := []string{
		"store_memory",
		"recall_memories",
		"replace_memory",
		"delete_memory",
	}
	for _, w := range want {
		if !containsTool(tools, w) {
			t.Fatalf("expected %s in tool list; got %d tools", w, len(tools))
		}
	}
}

//  4. Without memory configured at all: no recall injection, no extract
//     goroutine, no tool registration. Regression check.
func TestMemory_NoMemoryNoEffect(t *testing.T) {
	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithSystemPrompt("sys"),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if llmFake.sendMessageCallCount() != 0 {
		t.Fatalf("expected no SendMessages calls without memory; got %d",
			llmFake.sendMessageCallCount())
	}
	tools := llmFake.lastToolList()
	for _, name := range []string{"store_memory", "recall_memories"} {
		if containsTool(tools, name) {
			t.Fatalf("did not expect %s without memory configured", name)
		}
	}
}
