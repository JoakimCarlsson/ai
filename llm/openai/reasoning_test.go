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
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/types"
)

// TestExtractReasoningSendMessages tests extracting reasoning content from
// standard SendMessages non-streaming responses.
func TestExtractReasoningSendMessages(t *testing.T) {
	for _, key := range []string{"reasoning", "reasoning_content"} {
		t.Run("key_"+key, func(t *testing.T) {
			responseJSON := `{"id":"x","object":"chat.completion",` +
				`"choices":[{"index":0,"message":{"role":"assistant","content":"final answer",` +
				`"` + key + `":"thought process"},` +
				`"finish_reason":"stop"}],` +
				`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`

			srv := newCompletionServer(t, nil, responseJSON)
			defer srv.Close()

			client := NewLLM(
				WithAPIKey("test-key"),
				WithBaseURL(srv.URL),
				WithModel(model.Model{APIModel: "gpt-4o-mini"}),
			)

			resp, err := client.SendMessages(
				context.Background(),
				[]message.Message{message.NewUserMessage("hi")},
				nil,
			)
			if err != nil {
				t.Fatalf("SendMessages: %v", err)
			}

			if resp.Content != "final answer" {
				t.Errorf("Content = %q, want %q", resp.Content, "final answer")
			}
			if resp.Reasoning != "thought process" {
				t.Errorf(
					"Reasoning = %q, want %q",
					resp.Reasoning,
					"thought process",
				)
			}
		})
	}
}

// TestExtractReasoningStructuredOutput tests extracting reasoning content from
// SendMessagesWithStructuredOutput non-streaming responses.
func TestExtractReasoningStructuredOutput(t *testing.T) {
	responseJSON := `{"id":"x","object":"chat.completion",` +
		`"choices":[{"index":0,"message":{"role":"assistant","content":"final answer",` +
		`"reasoning_content":"structured thought"},` +
		`"finish_reason":"stop"}],` +
		`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`

	srv := newCompletionServer(t, nil, responseJSON)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	schemaInfo := &schema.StructuredOutputInfo{
		Name: "test_schema",
		Parameters: map[string]any{
			"type": "object",
		},
	}

	resp, err := client.SendMessagesWithStructuredOutput(
		context.Background(),
		[]message.Message{message.NewUserMessage("hi")},
		nil,
		schemaInfo,
	)
	if err != nil {
		t.Fatalf("SendMessagesWithStructuredOutput: %v", err)
	}

	if resp.Content != "final answer" {
		t.Errorf("Content = %q, want %q", resp.Content, "final answer")
	}
	if resp.Reasoning != "structured thought" {
		t.Errorf(
			"Reasoning = %q, want %q",
			resp.Reasoning,
			"structured thought",
		)
	}
}

// TestAccumulateReasoningStream tests streaming reasoning content events
// and checking the final compiled reasoning field.
func TestAccumulateReasoningStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			// Send first chunk with reasoning
			_, _ = io.WriteString(
				w,
				"data: {\"id\":\"x\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"thinking \"}}]}\n\n",
			)
			// Send second chunk with reasoning and content
			_, _ = io.WriteString(
				w,
				"data: {\"id\":\"x\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"more \",\"content\":\"hello\"}}]}\n\n",
			)
			// Send third chunk with content
			_, _ = io.WriteString(
				w,
				"data: {\"id\":\"x\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"}}]}\n\n",
			)
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	var events []llm.Event
	for evt := range client.StreamResponse(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil) {
		events = append(events, evt)
	}

	// Verify events
	var thinkingDeltas []string
	var finalResponse *llm.Response

	for _, evt := range events {
		switch evt.Type {
		case types.EventThinkingDelta:
			thinkingDeltas = append(thinkingDeltas, evt.Thinking)
		case types.EventComplete:
			finalResponse = evt.Response
		}
	}

	joinedThinking := strings.Join(thinkingDeltas, "")
	if joinedThinking != "thinking more " {
		t.Errorf(
			"Thinking deltas combined = %q, want %q",
			joinedThinking,
			"thinking more ",
		)
	}

	if finalResponse == nil {
		t.Fatal("Expected EventComplete event, but got none")
	}

	if finalResponse.Content != "hello world" {
		t.Errorf(
			"finalResponse.Content = %q, want %q",
			finalResponse.Content,
			"hello world",
		)
	}

	if finalResponse.Reasoning != "thinking more " {
		t.Errorf(
			"finalResponse.Reasoning = %q, want %q",
			finalResponse.Reasoning,
			"thinking more ",
		)
	}
}

// TestReplayReasoningContent tests that reasoning content is replayed
// when WithReasoningContentReplay(true) is configured.
func TestReplayReasoningContent(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithReasoningContentReplay(true),
	)

	msg := message.NewAssistantMessage()
	msg.AppendContent("final response text")
	msg.AppendReasoningContent("internal thought text")

	_, err := client.SendMessages(
		context.Background(),
		[]message.Message{msg},
		nil,
	)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	// Extract messages sent to the server
	msgs, ok := body["messages"].([]any)
	if !ok || len(msgs) == 0 {
		t.Fatalf("No messages in request body: %v", body)
	}

	assistantMsg, ok := msgs[0].(map[string]any)
	if !ok {
		t.Fatalf("First message is not map: %v", msgs[0])
	}

	if assistantMsg["role"] != "assistant" {
		t.Errorf("Role = %v, want assistant", assistantMsg["role"])
	}

	if assistantMsg["content"] != "final response text" {
		t.Errorf(
			"Content = %v, want %q",
			assistantMsg["content"],
			"final response text",
		)
	}

	if assistantMsg["reasoning_content"] != "internal thought text" {
		t.Errorf(
			"reasoning_content = %v, want %q",
			assistantMsg["reasoning_content"],
			"internal thought text",
		)
	}
}

// TestNoReplayReasoningContent tests that reasoning content is not replayed
// by default (WithReasoningContentReplay(false) or omitted).
func TestNoReplayReasoningContent(t *testing.T) {
	var body map[string]any
	srv := newCompletionServer(t, &body, completionOK)
	defer srv.Close()

	client := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
	)

	msg := message.NewAssistantMessage()
	msg.AppendContent("final response text")
	msg.AppendReasoningContent("internal thought text")

	_, err := client.SendMessages(
		context.Background(),
		[]message.Message{msg},
		nil,
	)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	// Extract messages sent to the server
	msgs, ok := body["messages"].([]any)
	if !ok || len(msgs) == 0 {
		t.Fatalf("No messages in request body: %v", body)
	}

	assistantMsg, ok := msgs[0].(map[string]any)
	if !ok {
		t.Fatalf("First message is not map: %v", msgs[0])
	}

	if assistantMsg["role"] != "assistant" {
		t.Errorf("Role = %v, want assistant", assistantMsg["role"])
	}

	if assistantMsg["content"] != "final response text" {
		t.Errorf(
			"Content = %v, want %q",
			assistantMsg["content"],
			"final response text",
		)
	}

	if val, exists := assistantMsg["reasoning_content"]; exists {
		t.Errorf("reasoning_content was set to %v, want it to be omitted", val)
	}
}
