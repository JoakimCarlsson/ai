package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// replayLLM decorates the tool response with a provider reasoning item.
type replayLLM struct {
	*mockLLM
	item message.ReasoningContent
}

// SendMessages includes encrypted reasoning alongside the first tool call.
func (m *replayLLM) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	resp, err := m.mockLLM.SendMessages(ctx, messages, tools)
	if err == nil && len(resp.ToolCalls) > 0 {
		resp.ReasoningItems = []message.ReasoningContent{m.item}
	}
	return resp, err
}

// StreamResponse includes the replayable item in the completed stream response.
func (m *replayLLM) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	out := make(chan llm.Event)
	go func() {
		defer close(out)
		for event := range m.mockLLM.StreamResponse(ctx, messages, tools) {
			if event.Response != nil && len(event.Response.ToolCalls) > 0 {
				event.Response.ReasoningItems = []message.ReasoningContent{
					m.item,
				}
			}
			out <- event
		}
	}()
	return out
}

// TestToolLoopPreservesReasoningItems verifies automatic tool execution retains
// encrypted reasoning in history for both Chat and ChatStream.
func TestToolLoopPreservesReasoningItems(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(
			map[bool]string{false: "chat", true: "stream"}[streaming],
			func(t *testing.T) {
				item := message.ReasoningContent{
					ID:               "rs_1",
					EncryptedContent: "opaque",
					Summary:          []string{"plan"},
					Field:            "reasoning",
					Details:          json.RawMessage(`[{"type":"reasoning.encrypted","data":"blob"}]`),
				}
				model := &replayLLM{mockLLM: newMockLLM(
					mockResponse{
						ToolCalls: []message.ToolCall{
							{
								ID:    "call_1",
								Name:  "echo",
								Input: `{"text":"hi"}`,
							},
						},
						FinishReason: message.FinishReasonToolUse,
					},
					mockResponse{Content: "done"},
				), item: item}
				a := agent.New(model, agent.WithTools(&echoTool{}))
				if streaming {
					for event := range a.ChatStream(context.Background(), "test") {
						if event.Type == types.EventError {
							t.Fatal(event.Error)
						}
					}
				} else if _, err := a.Chat(context.Background(), "test"); err != nil {
					t.Fatal(err)
				}
				if model.CallCount() != 2 {
					t.Fatalf("calls = %d", model.CallCount())
				}
				var assistant *message.Message
				var hasResult bool
				for i := range model.calls[1] {
					msg := &model.calls[1][i]
					if msg.Role == message.Assistant {
						assistant = msg
					}
					if msg.Role == message.Tool {
						hasResult = len(msg.ToolResults()) == 1
					}
				}
				if assistant == nil || !hasResult {
					t.Fatal("missing tool loop history")
				}
				if !reflect.DeepEqual(
					assistant.ReasoningContent(),
					[]message.ReasoningContent{item},
				) {
					t.Fatalf("reasoning = %+v", assistant.ReasoningContent())
				}
			},
		)
	}
}
