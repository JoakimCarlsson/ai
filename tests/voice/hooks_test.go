package voice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
	"github.com/joakimcarlsson/ai/voice"
)

// scriptComplete returns an LLM script that emits one delta and a complete
// event with no tool calls — clean turn end.
func scriptComplete(text string) func(ctx context.Context) <-chan llm.Event {
	return scriptedLLM(
		llm.Event{Type: types.EventContentDelta, Content: text},
		llm.Event{Type: types.EventComplete, Response: &llm.Response{}},
	)
}

// scriptOneTool returns an LLM script that emits a tool call then completes.
// The follow-up turn should be supplied separately by the caller.
func scriptOneTool(callID, name, input string) func(ctx context.Context) <-chan llm.Event {
	return scriptedLLM(
		llm.Event{Type: types.EventComplete, Response: &llm.Response{
			ToolCalls: []message.ToolCall{
				{ID: callID, Name: name, Input: input},
			},
		}},
	)
}

// 1. Deny on OnUserMessage: the LLM is never called and history stays empty.
func TestHooks_OnUserMessageDenyDropsTurn(t *testing.T) {
	llmFake := &fakeLLM{}
	a := newTestAgent(t, llmFake,
		voice.WithHooks(voice.Hooks{
			OnUserMessage: func(_ context.Context, _ voice.UserMessageContext) (voice.UserMessageResult, error) {
				return voice.UserMessageResult{
					Action:     voice.HookDeny,
					DenyReason: "blocked",
				}, nil
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("hello")

	waitFor(t, func() bool {
		for _, e := range a.eventsOfType(voice.EventError) {
			if errors.Is(e.Error, voice.ErrUserMessageDenied) {
				return true
			}
		}
		return false
	}, "deny event surfaced")

	if llmFake.callCount() != 0 {
		t.Fatalf("expected LLM not called after deny, got %d calls", llmFake.callCount())
	}

	a.cancel()
	_ = a.conv.Wait()
}

// 2. Modify on OnUserMessage: the modified text reaches the LLM.
func TestHooks_OnUserMessageModifyReplacesText(t *testing.T) {
	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithHooks(voice.Hooks{
			OnUserMessage: func(_ context.Context, uc voice.UserMessageContext) (voice.UserMessageResult, error) {
				return voice.UserMessageResult{
					Action: voice.HookModify,
					Text:   strings.ToUpper(uc.Text),
				}, nil
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("hello")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	msgs := llmFake.lastMessages()
	if !lastUserContains(msgs, "HELLO") {
		t.Fatalf("expected uppercased user text in LLM input; got %+v", msgs)
	}
}

// 3. PreModelCall modify: replaced messages reach the LLM.
func TestHooks_PreModelCallModifiesMessages(t *testing.T) {
	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	sentinel := message.NewUserMessage("INJECTED")

	a := newTestAgent(t, llmFake,
		voice.WithHooks(voice.Hooks{
			PreModelCall: func(_ context.Context, mc voice.ModelCallContext) (voice.ModelCallResult, error) {
				return voice.ModelCallResult{
					Action:   voice.HookModify,
					Messages: append(mc.Messages, sentinel),
					Tools:    mc.Tools,
				}, nil
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	msgs := llmFake.lastMessages()
	if !lastUserContains(msgs, "INJECTED") {
		t.Fatalf("expected injected message present; got %+v", msgs)
	}
}

// 4. PostModelCall fires exactly once per LLM iteration.
func TestHooks_PostModelCallObservesOnce(t *testing.T) {
	var calls int32
	var mu sync.Mutex

	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithHooks(voice.Hooks{
			PostModelCall: func(_ context.Context, _ voice.ModelResponseContext) {
				mu.Lock()
				calls++
				mu.Unlock()
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	mu.Lock()
	got := calls
	mu.Unlock()
	if got != 1 {
		t.Fatalf("expected PostModelCall fired once, got %d", got)
	}
}

// 5. PreToolUse deny: the tool is not invoked; history records the deny.
func TestHooks_PreToolUseDenySkipsToolExecution(t *testing.T) {
	tool := newFakeTool("badtool")
	llmFake := &fakeLLM{}
	llmFake.push(scriptOneTool("c1", "badtool", "{}"))
	llmFake.push(scriptComplete("done. "))

	a := newTestAgent(t, llmFake,
		voice.WithTools(tool),
		voice.WithHooks(voice.Hooks{
			PreToolUse: func(_ context.Context, _ voice.ToolUseContext) (voice.PreToolUseResult, error) {
				return voice.PreToolUseResult{
					Action:     voice.HookDeny,
					DenyReason: "policy",
				}, nil
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("call it")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if got := tool.lastReceivedInput(); got != "" {
		t.Fatalf("expected denied tool not invoked; saw input %q", got)
	}

	ends := a.eventsOfType(voice.EventToolCallEnd)
	if len(ends) == 0 {
		t.Fatalf("expected at least one EventToolCallEnd")
	}
	last := ends[len(ends)-1]
	if last.ToolResult == nil || !last.ToolResult.IsError ||
		!strings.Contains(last.ToolResult.Output, "policy") {
		t.Fatalf("expected tool-end with deny reason, got %+v", last.ToolResult)
	}
}

// 6. PreToolUse modify: the modified input reaches the tool.
func TestHooks_PreToolUseModifyChangesInput(t *testing.T) {
	tool := newFakeTool("mytool")
	llmFake := &fakeLLM{}
	llmFake.push(scriptOneTool("c1", "mytool", "original"))
	llmFake.push(scriptComplete("done. "))

	a := newTestAgent(t, llmFake,
		voice.WithTools(tool),
		voice.WithHooks(voice.Hooks{
			PreToolUse: func(_ context.Context, _ voice.ToolUseContext) (voice.PreToolUseResult, error) {
				return voice.PreToolUseResult{
					Action: voice.HookModify,
					Input:  "rewritten",
				}, nil
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("go")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if got := tool.lastReceivedInput(); got != "rewritten" {
		t.Fatalf("expected tool to receive 'rewritten', got %q", got)
	}
}

// 7. PostToolUse modify: the modified output is what the LLM sees.
func TestHooks_PostToolUseModifyReplacesOutput(t *testing.T) {
	tool := newFakeTool("mytool")
	tool.setOutput("raw")

	llmFake := &fakeLLM{}
	llmFake.push(scriptOneTool("c1", "mytool", "{}"))
	llmFake.push(scriptComplete("done. "))

	a := newTestAgent(t, llmFake,
		voice.WithTools(tool),
		voice.WithHooks(voice.Hooks{
			PostToolUse: func(_ context.Context, _ voice.PostToolUseContext) (voice.PostToolUseResult, error) {
				return voice.PostToolUseResult{
					Action: voice.HookModify,
					Output: "redacted",
				}, nil
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("call")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	msgs := llmFake.lastMessages()
	if !anyToolResultContains(msgs, "redacted") {
		t.Fatalf("expected tool result 'redacted' visible to LLM; got %+v", msgs)
	}
	if anyToolResultContains(msgs, "raw") {
		t.Fatalf("did not expect 'raw' to leak; got %+v", msgs)
	}
}

// 8. OnToolError modify: failed tool's output is replaced and IsError cleared.
func TestHooks_OnToolErrorRecoversWithFallbackOutput(t *testing.T) {
	tool := newFakeTool("flaky")
	tool.setError(errors.New("boom"))

	llmFake := &fakeLLM{}
	llmFake.push(scriptOneTool("c1", "flaky", "{}"))
	llmFake.push(scriptComplete("done. "))

	a := newTestAgent(t, llmFake,
		voice.WithTools(tool),
		voice.WithHooks(voice.Hooks{
			OnToolError: func(_ context.Context, _ voice.ToolErrorContext) (voice.ToolErrorResult, error) {
				return voice.ToolErrorResult{
					Action: voice.HookModify,
					Output: "fallback",
				}, nil
			},
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("go")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	ends := a.eventsOfType(voice.EventToolCallEnd)
	if len(ends) == 0 {
		t.Fatalf("expected at least one EventToolCallEnd")
	}
	last := ends[len(ends)-1]
	if last.ToolResult == nil || last.ToolResult.IsError {
		t.Fatalf("expected tool-end with IsError=false after recovery, got %+v", last.ToolResult)
	}
	if !strings.Contains(last.ToolResult.Output, "fallback") {
		t.Fatalf("expected fallback output, got %q", last.ToolResult.Output)
	}
}

// 9. Lifecycle hooks fire exactly once each.
func TestHooks_LifecycleFiresOnceEach(t *testing.T) {
	var startCount, endCount int32
	var mu sync.Mutex

	llmFake := &fakeLLM{}
	a := newTestAgent(t, llmFake,
		voice.WithHooks(voice.Hooks{
			OnConversationStart: func(_ context.Context, _ voice.ConversationLifecycleContext) {
				mu.Lock()
				startCount++
				mu.Unlock()
			},
			OnConversationEnd: func(_ context.Context, _ voice.ConversationLifecycleContext) {
				mu.Lock()
				endCount++
				mu.Unlock()
			},
		}),
	)

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return startCount == 1
	}, "OnConversationStart fired")

	a.cancel()
	_ = a.conv.Wait()

	mu.Lock()
	gotStart, gotEnd := startCount, endCount
	mu.Unlock()
	if gotStart != 1 {
		t.Fatalf("expected start fired once, got %d", gotStart)
	}
	if gotEnd != 1 {
		t.Fatalf("expected end fired once, got %d", gotEnd)
	}
}

// 10. Multiple hook structs run in registration order; modifies chain.
func TestHooks_MultipleHooksChainInOrder(t *testing.T) {
	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	first := voice.Hooks{
		OnUserMessage: func(_ context.Context, uc voice.UserMessageContext) (voice.UserMessageResult, error) {
			return voice.UserMessageResult{
				Action: voice.HookModify,
				Text:   uc.Text + "+A",
			}, nil
		},
	}
	second := voice.Hooks{
		OnUserMessage: func(_ context.Context, uc voice.UserMessageContext) (voice.UserMessageResult, error) {
			return voice.UserMessageResult{
				Action: voice.HookModify,
				Text:   uc.Text + "+B",
			}, nil
		},
	}

	a := newTestAgent(t, llmFake, voice.WithHooks(first, second))
	defer a.cancel()

	a.stt.pushFinal("X")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) }, "assistant done")

	a.cancel()
	_ = a.conv.Wait()

	msgs := llmFake.lastMessages()
	if !lastUserContains(msgs, "X+A+B") {
		t.Fatalf("expected chained 'X+A+B' in LLM input; got %+v", msgs)
	}
}

// --- helpers ---

func lastUserContains(msgs []message.Message, want string) bool {
	for _, m := range msgs {
		if m.Role != message.User {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(message.TextContent); ok &&
				strings.Contains(tc.Text, want) {
				return true
			}
		}
	}
	return false
}

func anyToolResultContains(msgs []message.Message, want string) bool {
	for _, m := range msgs {
		if m.Role != message.Tool {
			continue
		}
		for _, p := range m.Parts {
			if tr, ok := p.(message.ToolResult); ok &&
				strings.Contains(tr.Content, want) {
				return true
			}
		}
	}
	return false
}
