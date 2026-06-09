package gemini

import (
	"bytes"
	"testing"

	"github.com/joakimcarlsson/ai/message"
)

// TestGemini_ToolResultRoleIsUser verifies function/tool results are sent with
// role "user" (Gemini rejects role "function" with a 400 that kills the loop).
func TestGemini_ToolResultRoleIsUser(t *testing.T) {
	c := &Client{}
	toolMsg := message.NewMessage(message.Tool, nil)
	toolMsg.AddToolResult(
		message.ToolResult{ToolCallID: "1", Name: "search", Content: "result"},
	)

	contents, _ := c.convertMessages([]message.Message{toolMsg})
	if len(contents) == 0 {
		t.Fatalf("expected a tool-result content")
	}
	last := contents[len(contents)-1]
	if last.Role != "user" {
		t.Errorf("tool-result role = %q, want user", last.Role)
	}
	if len(last.Parts) == 0 || last.Parts[0].FunctionResponse == nil {
		t.Errorf("expected a FunctionResponse part on the tool-result content")
	}
}

// TestGemini_ThoughtSignatureReplay verifies a stored ToolCall.ThoughtSignature
// is replayed onto the outgoing function-call part so Gemini 3 accepts the turn.
func TestGemini_ThoughtSignatureReplay(t *testing.T) {
	sig := []byte("opaque-thought-sig")
	c := &Client{}
	asst := message.NewAssistantMessage()
	asst.AppendToolCalls([]message.ToolCall{{
		Name:             "search",
		Input:            "{}",
		Type:             "function",
		ThoughtSignature: sig,
	}})

	contents, _ := c.convertMessages([]message.Message{asst})
	var found bool
	for _, ct := range contents {
		for _, p := range ct.Parts {
			if p.FunctionCall != nil {
				found = true
				if !bytes.Equal(p.ThoughtSignature, sig) {
					t.Errorf(
						"replayed ThoughtSignature = %q, want %q",
						p.ThoughtSignature,
						sig,
					)
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected a function-call part in the converted messages")
	}
}

// TestGemini_NoThoughtSignatureIsClean verifies a tool call without a signature
// produces a nil/empty signature on the part (no spurious bytes).
func TestGemini_NoThoughtSignatureIsClean(t *testing.T) {
	c := &Client{}
	asst := message.NewAssistantMessage()
	asst.AppendToolCalls(
		[]message.ToolCall{{Name: "search", Input: "{}", Type: "function"}},
	)

	contents, _ := c.convertMessages([]message.Message{asst})
	for _, ct := range contents {
		for _, p := range ct.Parts {
			if p.FunctionCall != nil && len(p.ThoughtSignature) != 0 {
				t.Errorf(
					"expected empty ThoughtSignature, got %q",
					p.ThoughtSignature,
				)
			}
		}
	}
}
