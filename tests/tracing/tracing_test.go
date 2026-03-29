package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
		exporter.Reset()
	})
	return exporter
}

func findSpan(
	spans tracetest.SpanStubs,
	prefix string,
) *tracetest.SpanStub {
	for i, s := range spans {
		if len(s.Name) >= len(prefix) &&
			s.Name[:len(prefix)] == prefix {
			return &spans[i]
		}
	}
	return nil
}

func spanAttr(
	span *tracetest.SpanStub,
	key string,
) string {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			return attr.Value.Emit()
		}
	}
	return ""
}

func spanAttrInt(
	span *tracetest.SpanStub,
	key string,
) int64 {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			return attr.Value.AsInt64()
		}
	}
	return -1
}

func TestAgentChat_CreatesInvokeAgentSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(mockResponse{Content: "hello"})

	a := agent.New(mock)
	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span == nil {
		t.Fatal("expected invoke_agent span")
	}

	if spanAttr(span, "gen_ai.operation.name") != "invoke_agent" {
		t.Errorf(
			"expected operation.name 'invoke_agent', got %q",
			spanAttr(span, "gen_ai.operation.name"),
		)
	}
}

func TestAgentChat_CreatesChildSpans(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			Content: "",
			ToolCalls: []message.ToolCall{
				{ID: "tc1", Name: "echo", Input: `{"text":"hi"}`},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock, agent.WithTools(&echoTool{}))
	_, err := a.Chat(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}

	spans := exporter.GetSpans()

	agentSpan := findSpan(spans, "invoke_agent")
	toolSpan := findSpan(spans, "execute_tool")

	if agentSpan == nil {
		t.Fatal("expected invoke_agent span")
	}
	if toolSpan == nil {
		t.Fatal("expected execute_tool span")
	}

	if toolSpan.Parent.SpanID() != agentSpan.SpanContext.SpanID() {
		t.Error("execute_tool should be child of invoke_agent")
	}

	if spanAttr(toolSpan, "gen_ai.tool.name") != "echo" {
		t.Errorf(
			"expected tool name 'echo', got %q",
			spanAttr(toolSpan, "gen_ai.tool.name"),
		)
	}
	if spanAttr(toolSpan, "gen_ai.tool.call_id") != "tc1" {
		t.Errorf(
			"expected call_id 'tc1', got %q",
			spanAttr(toolSpan, "gen_ai.tool.call_id"),
		)
	}
}

func TestAgentChat_RecordsErrorOnSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{Err: fmt.Errorf("provider error")},
	)

	a := agent.New(mock)
	_, err := a.Chat(context.Background(), "hi")
	if err == nil {
		t.Fatal("expected error")
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span == nil {
		t.Fatal("expected invoke_agent span")
	}
	if span.Status.Code != codes.Error {
		t.Error("expected error status on invoke_agent span")
	}
}

func TestAgentChat_RecordsUsageAttrs(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(mockResponse{
		Content: "done",
		Usage: llm.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	})

	a := agent.New(mock)
	_, err := a.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}

	spans := exporter.GetSpans()
	span := findSpan(spans, "invoke_agent")
	if span == nil {
		t.Fatal("expected invoke_agent span")
	}

	if spanAttrInt(span, "gen_ai.usage.input_tokens") != 100 {
		t.Errorf(
			"expected input_tokens 100, got %d",
			spanAttrInt(span, "gen_ai.usage.input_tokens"),
		)
	}
	if spanAttrInt(span, "gen_ai.usage.output_tokens") != 50 {
		t.Errorf(
			"expected output_tokens 50, got %d",
			spanAttrInt(span, "gen_ai.usage.output_tokens"),
		)
	}
	if spanAttrInt(span, "gen_ai.agent.total_turns") != 1 {
		t.Errorf(
			"expected total_turns 1, got %d",
			spanAttrInt(span, "gen_ai.agent.total_turns"),
		)
	}
}

func TestAgentChatStream_CreatesSpans(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			Content: "",
			ToolCalls: []message.ToolCall{
				{ID: "tc1", Name: "echo", Input: `{"text":"hi"}`},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock, agent.WithTools(&echoTool{}))
	for evt := range a.ChatStream(context.Background(), "test") {
		_ = evt
	}

	spans := exporter.GetSpans()
	agentSpan := findSpan(spans, "invoke_agent")
	toolSpan := findSpan(spans, "execute_tool")

	if agentSpan == nil {
		t.Fatal("expected invoke_agent span")
	}
	if toolSpan == nil {
		t.Fatal("expected execute_tool span")
	}
}

func TestExecuteTool_DeniedByHook_NoToolSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{ID: "tc1", Name: "echo", Input: `{"text":"hi"}`},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{Content: "ok"},
	)

	a := agent.New(mock,
		agent.WithTools(&echoTool{}),
		agent.WithHooks(agent.Hooks{
			PreToolUse: func(
				_ context.Context,
				_ agent.ToolUseContext,
			) (agent.PreToolUseResult, error) {
				return agent.PreToolUseResult{
					Action:     agent.HookDeny,
					DenyReason: "blocked",
				}, nil
			},
		}),
	)
	_, _ = a.Chat(context.Background(), "test")

	spans := exporter.GetSpans()
	toolSpan := findSpan(spans, "execute_tool")
	if toolSpan != nil {
		t.Error("expected no execute_tool span when tool is denied by hook")
	}
}

type mockResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	FinishReason message.FinishReason
	Usage        llm.TokenUsage
	Err          error
}

type mockLLM struct {
	mu        sync.Mutex
	responses []mockResponse
	callIndex int
}

func newMockLLM(responses ...mockResponse) *mockLLM {
	return &mockLLM{responses: responses}
}

func (m *mockLLM) nextResponse() mockResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callIndex >= len(m.responses) {
		return mockResponse{Content: "no more responses"}
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp
}

func (m *mockLLM) SendMessages(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) (*llm.Response, error) {
	resp := m.nextResponse()
	if resp.Err != nil {
		return nil, resp.Err
	}
	return &llm.Response{
		Content:      resp.Content,
		ToolCalls:    resp.ToolCalls,
		FinishReason: resp.FinishReason,
		Usage:        resp.Usage,
	}, nil
}

func (m *mockLLM) SendMessagesWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *mockLLM) StreamResponse(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	go func() {
		defer close(ch)
		resp := m.nextResponse()
		if resp.Err != nil {
			ch <- llm.Event{
				Type:  types.EventError,
				Error: resp.Err,
			}
			return
		}
		ch <- llm.Event{
			Type: types.EventComplete,
			Response: &llm.Response{
				Content:      resp.Content,
				ToolCalls:    resp.ToolCalls,
				FinishReason: resp.FinishReason,
				Usage:        resp.Usage,
			},
		}
	}()
	return ch
}

func (m *mockLLM) StreamResponseWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *mockLLM) Model() model.Model {
	return model.Model{ID: "mock", Provider: "mock"}
}

func (m *mockLLM) SupportsStructuredOutput() bool {
	return false
}

type echoTool struct{}

func (t *echoTool) Info() tool.Info {
	return tool.NewInfo("echo", "Echoes input", struct {
		Text string `json:"text" desc:"Text to echo"`
	}{})
}

func (t *echoTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(params.Input), &input)
	return tool.NewTextResponse("echo: " + input.Text), nil
}
