package agent

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/message"
)

func TestExecuteTool_CreatesSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc1",
					Name:  "echo",
					Input: `{"text":"hi"}`,
				},
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
	toolSpan := findSpan(spans, "execute_tool")
	if toolSpan == nil {
		t.Fatal("expected execute_tool span")
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

func TestExecuteTool_DeniedByHook_NoSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc1",
					Name:  "echo",
					Input: `{"text":"hi"}`,
				},
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
	if findSpan(spans, "execute_tool") != nil {
		t.Error(
			"expected no execute_tool span when denied by hook",
		)
	}
}

func TestMergedToolSpan_MultipleTools(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc1",
					Name:  "echo",
					Input: `{"text":"a"}`,
				},
				{
					ID:    "tc2",
					Name:  "echo",
					Input: `{"text":"b"}`,
				},
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
	mergedSpan := findSpan(spans, "execute_tools")
	if mergedSpan == nil {
		t.Fatal("expected execute_tools merged span")
	}
	if spanAttrInt(
		mergedSpan,
		"gen_ai.request.tool_count",
	) != 2 {
		t.Errorf(
			"expected tool_count 2, got %d",
			spanAttrInt(mergedSpan, "gen_ai.request.tool_count"),
		)
	}

	var toolSpanCount int
	for _, s := range spans {
		if len(s.Name) >= len("execute_tool ") &&
			s.Name[:len("execute_tool ")] == "execute_tool " {
			if s.Parent.SpanID() !=
				mergedSpan.SpanContext.SpanID() {
				t.Error(
					"execute_tool should be child of execute_tools",
				)
			}
			toolSpanCount++
		}
	}
	if toolSpanCount != 2 {
		t.Errorf(
			"expected 2 execute_tool spans, got %d",
			toolSpanCount,
		)
	}
}

func TestMergedToolSpan_SingleTool_NoMergedSpan(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc1",
					Name:  "echo",
					Input: `{"text":"a"}`,
				},
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
	if findSpan(spans, "execute_tools") != nil {
		t.Error(
			"expected no execute_tools span for single tool",
		)
	}
	if findSpan(spans, "execute_tool") == nil {
		t.Fatal("expected execute_tool span")
	}
}

func TestMergedToolSpan_Stream(t *testing.T) {
	exporter := setupTracing(t)
	mock := newMockLLM(
		mockResponse{
			ToolCalls: []message.ToolCall{
				{
					ID:    "tc1",
					Name:  "echo",
					Input: `{"text":"a"}`,
				},
				{
					ID:    "tc2",
					Name:  "echo",
					Input: `{"text":"b"}`,
				},
			},
			FinishReason: message.FinishReasonToolUse,
		},
		mockResponse{Content: "done"},
	)

	a := agent.New(mock, agent.WithTools(&echoTool{}))
	for evt := range a.ChatStream(
		context.Background(),
		"test",
	) {
		_ = evt
	}

	spans := exporter.GetSpans()
	if findSpan(spans, "execute_tools") == nil {
		t.Fatal(
			"expected execute_tools merged span in streaming",
		)
	}
}
