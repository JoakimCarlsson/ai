package voice

import (
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/voice"
)

// makeHandoffAgent constructs a target VoiceAgent for handoff with its own
// fakeLLM and dummy STT/TTS (the latter are ignored after handoff).
func makeHandoffAgent(
	id, prompt string,
	opts ...voice.Option,
) (*voice.Agent, *fakeLLM) {
	lf := newFakeLLM(id)
	stf := newFakeSTT()
	tts := newFakeTTS(true)
	all := append(
		[]voice.Option{voice.WithSystemPrompt(prompt)},
		opts...,
	)
	return voice.New(lf, stf, tts, all...), lf
}

// 1. Registering a handoff exposes a transfer_to_<Name> tool to the LLM.
func TestHandoff_RegistersTransferTool(t *testing.T) {
	billing, _ := makeHandoffAgent("billing", "billing")

	triage := newFakeLLM("triage")
	triage.push(scriptComplete("ok. "))

	a := newTestAgent(t, triage,
		voice.WithSystemPrompt("triage"),
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "billing",
			Description: "billing",
			Agent:       billing,
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("hi")
	waitFor(t, func() bool { return a.hasEvent(voice.EventAssistantDone) },
		"assistant done")

	a.cancel()
	_ = a.conv.Wait()

	tools := triage.lastToolList()
	if !containsTool(tools, "transfer_to_billing") {
		t.Fatalf(
			"expected transfer_to_billing in LLM tool list; got %d tools",
			len(tools),
		)
	}
}

//  2. Calling the handoff tool switches the active agent: the next LLM call
//     goes to the target's fakeLLM, not the source.
func TestHandoff_SwitchesActiveAgentAfterToolCall(t *testing.T) {
	billing, billingLLM := makeHandoffAgent("billing", "you are billing")
	billingLLM.push(scriptComplete("hi from billing. "))

	triage := newFakeLLM("triage")
	triage.push(
		scriptOneTool("c1", "transfer_to_billing", `{"reason":"refund"}`),
	)

	a := newTestAgent(t, triage,
		voice.WithSystemPrompt("you are triage"),
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "billing",
			Description: "for billing",
			Agent:       billing,
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("refund please")
	waitFor(t, func() bool { return billingLLM.callCount() == 1 },
		"billing LLM called once")

	a.cancel()
	_ = a.conv.Wait()

	if triage.callCount() != 1 {
		t.Fatalf("expected triage called once, got %d", triage.callCount())
	}
}

//  3. After handoff the new system prompt is the only system message in the
//     list the target LLM receives.
func TestHandoff_RebuildsHistoryWithNewSystemPrompt(t *testing.T) {
	billing, billingLLM := makeHandoffAgent("billing", "BILLING_PROMPT")
	billingLLM.push(scriptComplete("ok. "))

	triage := newFakeLLM("triage")
	triage.push(scriptOneTool("c1", "transfer_to_billing", `{}`))

	a := newTestAgent(t, triage,
		voice.WithSystemPrompt("TRIAGE_PROMPT"),
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "billing",
			Description: "for billing",
			Agent:       billing,
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("transfer me")
	waitFor(t, func() bool { return billingLLM.callCount() == 1 },
		"billing LLM called")

	a.cancel()
	_ = a.conv.Wait()

	msgs := billingLLM.lastMessages()
	systemCount := 0
	var systemText string
	for _, m := range msgs {
		if m.Role == message.System {
			systemCount++
			for _, p := range m.Parts {
				if tc, ok := p.(message.TextContent); ok {
					systemText = tc.Text
				}
			}
		}
	}
	if systemCount != 1 {
		t.Fatalf(
			"expected exactly 1 system message in billing LLM input, got %d: %+v",
			systemCount,
			msgs,
		)
	}
	if systemText != "BILLING_PROMPT" {
		t.Fatalf("expected billing prompt, got %q", systemText)
	}
}

// 4. Pre-handoff user/assistant/tool messages are preserved across the swap.
func TestHandoff_PreservesNonSystemHistory(t *testing.T) {
	billing, billingLLM := makeHandoffAgent("billing", "billing")
	billingLLM.push(scriptComplete("ok. "))

	triage := newFakeLLM("triage")
	triage.push(scriptOneTool("c1", "transfer_to_billing", `{}`))

	a := newTestAgent(t, triage,
		voice.WithSystemPrompt("triage"),
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "billing",
			Description: "for billing",
			Agent:       billing,
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("the user question")
	waitFor(t, func() bool { return billingLLM.callCount() == 1 },
		"billing LLM called")

	a.cancel()
	_ = a.conv.Wait()

	msgs := billingLLM.lastMessages()
	if !lastUserContains(msgs, "the user question") {
		t.Fatalf(
			"expected user message preserved across handoff; got %+v",
			msgs,
		)
	}
	// The transfer tool call's message and result should also have been
	// preserved as part of history (assistant tool-call + tool result).
	var sawToolResult bool
	for _, m := range msgs {
		if m.Role != message.Tool {
			continue
		}
		for _, p := range m.Parts {
			if tr, ok := p.(message.ToolResult); ok &&
				strings.Contains(tr.Content, "Transferring to billing") {
				sawToolResult = true
			}
		}
	}
	if !sawToolResult {
		t.Fatalf("expected handoff tool result preserved; got %+v", msgs)
	}
}

// 5. Chained handoffs A→B→C with a single user turn.
func TestHandoff_ChainedAtoBtoC(t *testing.T) {
	cAgent, cLLM := makeHandoffAgent("c", "C_PROMPT")
	cLLM.push(scriptComplete("done from c. "))

	bAgent, bLLM := makeHandoffAgent("b", "B_PROMPT",
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "c",
			Description: "to c",
			Agent:       cAgent,
		}),
	)
	bLLM.push(scriptOneTool("c2", "transfer_to_c", `{}`))

	aLLM := newFakeLLM("a")
	aLLM.push(scriptOneTool("c1", "transfer_to_b", `{}`))

	a := newTestAgent(t, aLLM,
		voice.WithSystemPrompt("A_PROMPT"),
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "b",
			Description: "to b",
			Agent:       bAgent,
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("start")
	waitFor(t, func() bool { return cLLM.callCount() == 1 },
		"C LLM called once")

	a.cancel()
	_ = a.conv.Wait()

	if aLLM.callCount() != 1 {
		t.Fatalf("expected A called once, got %d", aLLM.callCount())
	}
	if bLLM.callCount() != 1 {
		t.Fatalf("expected B called once, got %d", bLLM.callCount())
	}
}

//  6. The handoff tool's "Transferring to X" response (or with reason) lands
//     in history.
func TestHandoff_ToolMessageRecordsTransferring(t *testing.T) {
	billing, billingLLM := makeHandoffAgent("billing", "billing")
	billingLLM.push(scriptComplete("ok. "))

	triage := newFakeLLM("triage")
	triage.push(
		scriptOneTool(
			"c1",
			"transfer_to_billing",
			`{"reason":"customer wants refund"}`,
		),
	)

	a := newTestAgent(t, triage,
		voice.WithSystemPrompt("triage"),
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "billing",
			Description: "for billing",
			Agent:       billing,
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("refund")
	waitFor(t, func() bool { return billingLLM.callCount() == 1 },
		"billing LLM called")

	a.cancel()
	_ = a.conv.Wait()

	msgs := billingLLM.lastMessages()
	var found bool
	for _, m := range msgs {
		if m.Role != message.Tool {
			continue
		}
		for _, p := range m.Parts {
			if tr, ok := p.(message.ToolResult); ok &&
				strings.Contains(tr.Content, "Transferring to billing") &&
				strings.Contains(tr.Content, "customer wants refund") {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected transferring-with-reason tool result, got %+v", msgs)
	}
}

// 7. Subsequent user turns continue with the post-handoff agent.
func TestHandoff_NextUserTurnStaysWithNewAgent(t *testing.T) {
	billing, billingLLM := makeHandoffAgent("billing", "billing")
	// Two scripts: one for the immediate post-handoff reply, one for
	// the next user turn after that.
	billingLLM.push(scriptComplete("first billing reply. "))
	billingLLM.push(scriptComplete("second billing reply. "))

	triage := newFakeLLM("triage")
	triage.push(scriptOneTool("c1", "transfer_to_billing", `{}`))

	a := newTestAgent(t, triage,
		voice.WithSystemPrompt("triage"),
		voice.WithHandoffs(voice.HandoffConfig{
			Name:        "billing",
			Description: "for billing",
			Agent:       billing,
		}),
	)
	defer a.cancel()

	a.stt.pushFinal("transfer me")
	waitFor(t, func() bool { return billingLLM.callCount() == 1 },
		"billing LLM called once for handoff continuation")

	a.stt.pushFinal("now another question")
	waitFor(t, func() bool { return billingLLM.callCount() == 2 },
		"billing LLM called for the next user turn too")

	a.cancel()
	_ = a.conv.Wait()

	if triage.callCount() != 1 {
		t.Fatalf("expected triage called only once, got %d", triage.callCount())
	}
}

// containsTool returns true when tools includes a tool whose Info().Name
// matches want.
func containsTool(tools []tool.BaseTool, want string) bool {
	for _, t := range tools {
		if t.Info().Name == want {
			return true
		}
	}
	return false
}
