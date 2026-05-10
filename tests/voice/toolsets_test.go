package voice

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/voice"
)

// dynamicToolset is a Toolset whose Tools(ctx) result depends on a counter
// or context value, used to verify per-call evaluation.
type dynamicToolset struct {
	name  string
	build func(ctx context.Context) []tool.BaseTool
}

func (d *dynamicToolset) Name() string { return d.name }

func (d *dynamicToolset) Tools(
	ctx context.Context,
) []tool.BaseTool {
	return d.build(ctx)
}

// 1. Toolset-resolved tools reach the LLM alongside static tools.
func TestToolsets_ToolsReachLLM(t *testing.T) {
	staticT := newFakeTool("static_tool")
	dynT := newFakeTool("dyn_tool")
	ts := &dynamicToolset{
		name:  "dynset",
		build: func(_ context.Context) []tool.BaseTool { return []tool.BaseTool{dynT} },
	}

	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("ok. "))

	a := newTestAgent(t, llmFake,
		voice.WithTools(staticT),
		voice.WithToolsets(ts),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	tools := llmFake.lastToolList()
	if !containsTool(tools, "static_tool") {
		t.Fatalf("expected static_tool in LLM tool list; got %d", len(tools))
	}
	if !containsTool(tools, "dyn_tool") {
		t.Fatalf(
			"expected dyn_tool from toolset in LLM tool list; got %d",
			len(tools),
		)
	}
}

//  2. The LLM can invoke a toolset-resolved tool — findTool consults the
//     union, not just static tools.
func TestToolsets_LLMCanInvokeToolsetTool(t *testing.T) {
	dynT := newFakeTool("dyn_tool")
	dynT.setOutput("dyn_result")
	ts := &dynamicToolset{
		name:  "dynset",
		build: func(_ context.Context) []tool.BaseTool { return []tool.BaseTool{dynT} },
	}

	llmFake := &fakeLLM{}
	llmFake.push(scriptOneTool("c1", "dyn_tool", `{}`))
	llmFake.push(scriptComplete("done. "))

	a := newTestAgent(t, llmFake,
		voice.WithToolsets(ts),
	)
	defer a.cancel()

	a.stt.pushFinal("call dyn")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if dynT.lastReceivedInput() != "{}" {
		t.Fatalf(
			"expected dyn_tool invoked; lastInput=%q",
			dynT.lastReceivedInput(),
		)
	}
	ends := a.eventsOfType(voice.EventToolCallEnd)
	if len(ends) == 0 {
		t.Fatalf("expected EventToolCallEnd")
	}
	if ends[0].ToolResult == nil ||
		ends[0].ToolResult.Output != "dyn_result" {
		t.Fatalf("expected dyn_result output, got %+v", ends[0].ToolResult)
	}
}

//  3. Toolset.Tools(ctx) is evaluated per LLM call (not cached at agent
//     construction). Each conversation turn re-resolves the set.
func TestToolsets_EvaluatedPerCall(t *testing.T) {
	var calls int32
	t1 := newFakeTool("t1")
	ts := &dynamicToolset{
		name: "counting",
		build: func(_ context.Context) []tool.BaseTool {
			atomic.AddInt32(&calls, 1)
			return []tool.BaseTool{t1}
		},
	}

	llmFake := &fakeLLM{}
	llmFake.push(scriptComplete("first. "))
	llmFake.push(scriptComplete("second. "))

	a := newTestAgent(t, llmFake,
		voice.WithToolsets(ts),
	)
	defer a.cancel()

	a.stt.pushFinal("first turn")
	waitFor(
		t,
		func() bool { return a.countEvents(voice.EventAssistantDone) == 1 },
		"first turn done",
	)

	a.stt.pushFinal("second turn")
	waitFor(
		t,
		func() bool { return a.countEvents(voice.EventAssistantDone) == 2 },
		"second turn done",
	)

	a.cancel()
	_ = a.conv.Wait()

	got := atomic.LoadInt32(&calls)
	if got < 2 {
		t.Fatalf(
			"expected toolset Tools() called >= 2 times across 2 turns, got %d",
			got,
		)
	}
}

//  4. Static tools and toolset tools are both invokable in the same turn —
//     findTool resolves names from the union.
func TestToolsets_StaticAndDynamicCoexist(t *testing.T) {
	staticT := newFakeTool("static_tool")
	dynT := newFakeTool("dyn_tool")
	ts := &dynamicToolset{
		name:  "dynset",
		build: func(_ context.Context) []tool.BaseTool { return []tool.BaseTool{dynT} },
	}

	llmFake := &fakeLLM{}
	// First iter: call dyn_tool. Second iter: call static_tool. Third: done.
	llmFake.push(scriptOneTool("c1", "dyn_tool", `{}`))
	llmFake.push(scriptOneTool("c2", "static_tool", `{}`))
	llmFake.push(scriptComplete("done. "))

	a := newTestAgent(t, llmFake,
		voice.WithTools(staticT),
		voice.WithToolsets(ts),
	)
	defer a.cancel()

	a.stt.pushFinal("go")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	if dynT.lastReceivedInput() == "" {
		t.Fatalf("expected dyn_tool invoked")
	}
	if staticT.lastReceivedInput() == "" {
		t.Fatalf("expected static_tool invoked")
	}
}
