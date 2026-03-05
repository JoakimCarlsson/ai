package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/types"
)

func TestSupportsStructuredOutput(t *testing.T) {
	supported := &anthropicClient{
		llmOptions: llmClientOptions{
			model: model.AnthropicModels[model.Claude45Sonnet],
		},
	}
	if !supported.supportsStructuredOutput() {
		t.Error("expected Claude45Sonnet to support structured output")
	}

	unsupported := &anthropicClient{
		llmOptions: llmClientOptions{
			model: model.AnthropicModels[model.Claude35Sonnet],
		},
	}
	if unsupported.supportsStructuredOutput() {
		t.Error("expected Claude35Sonnet to NOT support structured output")
	}
}

func TestBuildOutputConfig(t *testing.T) {
	client := &anthropicClient{}
	outputSchema := schema.NewStructuredOutputInfo(
		"test_schema", "A test schema",
		map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "integer"},
		},
		[]string{"name"},
	)

	config := client.buildOutputConfig(outputSchema)

	schemaMap := config.Format.Schema
	if schemaMap["type"] != "object" {
		t.Errorf("expected type=object, got %v", schemaMap["type"])
	}
	if schemaMap["properties"] == nil {
		t.Error("expected properties to be set")
	}
	required, ok := schemaMap["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required=[name], got %v", schemaMap["required"])
	}
}

func TestBuildOutputConfigNoRequired(t *testing.T) {
	client := &anthropicClient{}
	outputSchema := schema.NewStructuredOutputInfo(
		"test", "test",
		map[string]any{"name": map[string]any{"type": "string"}},
		nil,
	)

	config := client.buildOutputConfig(outputSchema)
	if _, exists := config.Format.Schema["required"]; exists {
		t.Error("expected no required field when Required is nil")
	}
}

func TestSendWithStructuredOutput(t *testing.T) {
	jsonResponse := `{"name":"Alice","age":30}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if _, ok := body["output_config"]; !ok {
			t.Error("expected output_config in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-5-20250929",
			"content": []map[string]any{
				{"type": "text", "text": jsonResponse},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  10,
				"output_tokens": 20,
			},
		})
	}))
	defer server.Close()

	client := &anthropicClient{
		llmOptions: llmClientOptions{
			model:     model.AnthropicModels[model.Claude45Sonnet],
			maxTokens: 1024,
		},
		client: anthropic.NewClient(
			option.WithBaseURL(server.URL),
			option.WithAPIKey("test-key"),
		),
	}

	outputSchema := schema.NewStructuredOutputInfo(
		"person", "A person",
		map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "integer"},
		},
		[]string{"name", "age"},
	)

	resp, err := client.sendWithStructuredOutput(
		context.Background(),
		[]message.Message{message.NewUserMessage("Return a person")},
		nil,
		outputSchema,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StructuredOutput == nil {
		t.Fatal("expected StructuredOutput to be non-nil")
	}
	if *resp.StructuredOutput != jsonResponse {
		t.Errorf("expected %q, got %q", jsonResponse, *resp.StructuredOutput)
	}
	if !resp.UsedNativeStructuredOutput {
		t.Error("expected UsedNativeStructuredOutput to be true")
	}
}

func TestStreamWithStructuredOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher := w.(http.Flusher)

		events := []string{
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","model":"claude-sonnet-4-5-20250929","content":[],"stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"{\"name\":"}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"\"Alice\",\"age\":30}"}}`,
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":0}`,
			`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":15}}`,
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
		}

		for _, event := range events {
			fmt.Fprintf(w, "%s\n\n", event)
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := &anthropicClient{
		llmOptions: llmClientOptions{
			model:     model.AnthropicModels[model.Claude45Sonnet],
			maxTokens: 1024,
		},
		client: anthropic.NewClient(
			option.WithBaseURL(server.URL),
			option.WithAPIKey("test-key"),
		),
	}

	outputSchema := schema.NewStructuredOutputInfo(
		"person", "A person",
		map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "integer"},
		},
		[]string{"name", "age"},
	)

	eventChan := client.streamWithStructuredOutput(
		context.Background(),
		[]message.Message{message.NewUserMessage("Return a person")},
		nil,
		outputSchema,
	)

	var gotContentDelta bool
	var finalResponse *LLMResponse

	for event := range eventChan {
		switch event.Type {
		case types.EventContentDelta:
			gotContentDelta = true
		case types.EventComplete:
			finalResponse = event.Response
		case types.EventError:
			t.Fatalf("unexpected error event: %v", event.Error)
		}
	}

	if !gotContentDelta {
		t.Error("expected at least one content delta event")
	}
	if finalResponse == nil {
		t.Fatal("expected a complete event with response")
	}
	if finalResponse.StructuredOutput == nil {
		t.Fatal("expected StructuredOutput to be non-nil")
	}
	if !finalResponse.UsedNativeStructuredOutput {
		t.Error("expected UsedNativeStructuredOutput to be true")
	}

	expected := `{"name":"Alice","age":30}`
	if *finalResponse.StructuredOutput != expected {
		t.Errorf("expected structured output %q, got %q", expected, *finalResponse.StructuredOutput)
	}
}
