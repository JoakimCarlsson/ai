package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/types"
)

// newChunkServer returns a test server that replies to every request with the
// given chat-completion chunk payloads as a server-sent event stream, followed
// by the [DONE] sentinel.
func newChunkServer(t *testing.T, chunks ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, chunk := range chunks {
				_, _ = io.WriteString(w, "data: "+chunk+"\n\n")
			}
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
}

// choiceChunk builds one chat-completion chunk carrying a single choice with
// the given index, delta object and finish reason. An empty finishReason is
// sent as null.
func choiceChunk(index, delta, finishReason string) string {
	finish := "null"
	if finishReason != "" {
		finish = `"` + finishReason + `"`
	}
	return `{"id":"x","object":"chat.completion.chunk","choices":[{"index":` +
		index + `,"delta":` + delta + `,"finish_reason":` + finish + `}]}`
}

// collectStream drains a streaming request against srv and returns the
// concatenated content deltas, the concatenated thinking deltas and the final
// response. It fails the test on an error event or a missing final response.
func collectStream(
	t *testing.T,
	srv *httptest.Server,
	opts ...Option,
) (string, string, *llm.Response) {
	t.Helper()
	client := NewChatCompletionsLLM(append([]Option{
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(llm.Model{APIModel: "gpt-4o-mini"}),
	}, opts...)...)

	var content, thinking strings.Builder
	var final *llm.Response
	for evt := range client.StreamResponse(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil) {
		switch evt.Type {
		case types.EventContentDelta:
			content.WriteString(evt.Content)
		case types.EventThinkingDelta:
			thinking.WriteString(evt.Thinking)
		case types.EventComplete:
			final = evt.Response
		case types.EventError:
			t.Fatalf("stream error: %v", evt.Error)
		}
	}
	if final == nil {
		t.Fatal("stream ended without a final response")
	}
	return content.String(), thinking.String(), final
}

// TestStreamMultipleChoicesIsolated confirms that two interleaved streamed
// choices are kept apart: deltas and the top-level response fields carry only
// choice 0, while Response.Choices holds every choice with its own content,
// finish reason and tool calls.
func TestStreamMultipleChoicesIsolated(t *testing.T) {
	srv := newChunkServer(t,
		choiceChunk("0", `{"role":"assistant","reasoning_content":"think0 "}`, ""),
		choiceChunk("1", `{"role":"assistant","reasoning_content":"think1 "}`, ""),
		choiceChunk("0", `{"content":"first "}`, ""),
		choiceChunk("1", `{"content":"second "}`, ""),
		choiceChunk("1", `{"tool_calls":[{"index":0,"id":"call_1",`+
			`"type":"function","function":{"name":"lookup",`+
			`"arguments":"{\"q\":"}}]}`, ""),
		choiceChunk("0", `{"content":"answer"}`, ""),
		choiceChunk("1", `{"tool_calls":[{"index":0,`+
			`"function":{"arguments":"\"x\"}"}}]}`, ""),
		choiceChunk("1", `{}`, "tool_calls"),
		choiceChunk("0", `{}`, "stop"),
	)
	defer srv.Close()

	content, thinking, resp := collectStream(t, srv, WithN(2))

	if content != "first answer" {
		t.Errorf("content deltas = %q, want %q", content, "first answer")
	}
	if thinking != "think0 " {
		t.Errorf("thinking deltas = %q, want %q", thinking, "think0 ")
	}
	if resp.Content != "first answer" {
		t.Errorf("Content = %q, want %q", resp.Content, "first answer")
	}
	if resp.Reasoning != "think0 " {
		t.Errorf("Reasoning = %q, want %q", resp.Reasoning, "think0 ")
	}
	if resp.FinishReason != message.FinishReasonEndTurn {
		t.Errorf("FinishReason = %q, want %q",
			resp.FinishReason, message.FinishReasonEndTurn)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls = %v, want none for choice 0", resp.ToolCalls)
	}

	if len(resp.Choices) != 2 {
		t.Fatalf("len(Choices) = %d, want 2", len(resp.Choices))
	}
	if resp.Choices[0].Content != "first answer" {
		t.Errorf("Choices[0].Content = %q, want %q",
			resp.Choices[0].Content, "first answer")
	}
	second := resp.Choices[1]
	if second.Content != "second " {
		t.Errorf("Choices[1].Content = %q, want %q", second.Content, "second ")
	}
	if second.FinishReason != message.FinishReasonToolUse {
		t.Errorf("Choices[1].FinishReason = %q, want %q",
			second.FinishReason, message.FinishReasonToolUse)
	}
	if len(second.ToolCalls) != 1 {
		t.Fatalf("len(Choices[1].ToolCalls) = %d, want 1",
			len(second.ToolCalls))
	}
	if second.ToolCalls[0].Name != "lookup" ||
		second.ToolCalls[0].Input != `{"q":"x"}` {
		t.Errorf("Choices[1].ToolCalls[0] = %+v, want lookup with {\"q\":\"x\"}",
			second.ToolCalls[0])
	}
}

// TestStreamSingleChoiceUnchanged is the single-choice control: every delta is
// emitted and Response.Choices stays empty.
func TestStreamSingleChoiceUnchanged(t *testing.T) {
	srv := newChunkServer(t,
		choiceChunk("0", `{"role":"assistant","content":"hello"}`, ""),
		choiceChunk("0", `{"content":" world"}`, ""),
		choiceChunk("0", `{}`, "stop"),
	)
	defer srv.Close()

	content, _, resp := collectStream(t, srv)

	if content != "hello world" || resp.Content != "hello world" {
		t.Errorf("deltas = %q, Content = %q, want %q for both",
			content, resp.Content, "hello world")
	}
	if len(resp.Choices) != 0 {
		t.Errorf("Choices = %v, want empty for a single choice", resp.Choices)
	}
}
