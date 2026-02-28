package test

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

func makeCall(name, input string) tool.ToolCall {
	return tool.ToolCall{ID: "test-1", Name: name, Input: input}
}

type mockResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	FinishReason message.FinishReason
	Err          error
}

type mockLLM struct {
	mu        sync.Mutex
	responses []mockResponse
	callIndex int
	calls     [][]message.Message
}

func newMockLLM(responses ...mockResponse) *mockLLM {
	return &mockLLM{responses: responses}
}

func (m *mockLLM) nextResponse() mockResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callIndex >= len(m.responses) {
		return mockResponse{Content: "no more responses configured"}
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp
}

func (m *mockLLM) recordCall(msgs []message.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, msgs)
}

func (m *mockLLM) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockLLM) SendMessages(_ context.Context, msgs []message.Message, _ []tool.BaseTool) (*llm.LLMResponse, error) {
	m.recordCall(msgs)
	resp := m.nextResponse()
	if resp.Err != nil {
		return nil, resp.Err
	}
	return &llm.LLMResponse{
		Content:      resp.Content,
		ToolCalls:    resp.ToolCalls,
		FinishReason: resp.FinishReason,
	}, nil
}

func (m *mockLLM) SendMessagesWithStructuredOutput(_ context.Context, _ []message.Message, _ []tool.BaseTool, _ *schema.StructuredOutputInfo) (*llm.LLMResponse, error) {
	return nil, nil
}

func (m *mockLLM) StreamResponse(_ context.Context, msgs []message.Message, _ []tool.BaseTool) <-chan llm.LLMEvent {
	m.recordCall(msgs)
	ch := make(chan llm.LLMEvent)
	go func() {
		defer close(ch)
		resp := m.nextResponse()
		if resp.Err != nil {
			ch <- llm.LLMEvent{Type: types.EventError, Error: resp.Err}
			return
		}
		if resp.Content != "" {
			ch <- llm.LLMEvent{Type: types.EventContentDelta, Content: resp.Content}
		}
		ch <- llm.LLMEvent{
			Type: types.EventComplete,
			Response: &llm.LLMResponse{
				Content:      resp.Content,
				ToolCalls:    resp.ToolCalls,
				FinishReason: resp.FinishReason,
			},
		}
	}()
	return ch
}

func (m *mockLLM) StreamResponseWithStructuredOutput(_ context.Context, _ []message.Message, _ []tool.BaseTool, _ *schema.StructuredOutputInfo) <-chan llm.LLMEvent {
	ch := make(chan llm.LLMEvent)
	close(ch)
	return ch
}

func (m *mockLLM) Model() model.Model {
	return model.Model{ID: "mock-model", Provider: "mock"}
}

func (m *mockLLM) SupportsStructuredOutput() bool {
	return false
}

type concurrencyTrackingLLM struct {
	base              *mockLLM
	maxConcurrent     *atomic.Int32
	currentConcurrent *atomic.Int32
	delay             time.Duration
}

func (m *concurrencyTrackingLLM) SendMessages(ctx context.Context, msgs []message.Message, tools []tool.BaseTool) (*llm.LLMResponse, error) {
	cur := m.currentConcurrent.Add(1)
	defer m.currentConcurrent.Add(-1)

	for {
		old := m.maxConcurrent.Load()
		if cur <= old {
			break
		}
		if m.maxConcurrent.CompareAndSwap(old, cur) {
			break
		}
	}

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	return m.base.SendMessages(ctx, msgs, tools)
}

func (m *concurrencyTrackingLLM) SendMessagesWithStructuredOutput(ctx context.Context, msgs []message.Message, tools []tool.BaseTool, info *schema.StructuredOutputInfo) (*llm.LLMResponse, error) {
	return m.base.SendMessagesWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *concurrencyTrackingLLM) StreamResponse(ctx context.Context, msgs []message.Message, tools []tool.BaseTool) <-chan llm.LLMEvent {
	return m.base.StreamResponse(ctx, msgs, tools)
}

func (m *concurrencyTrackingLLM) StreamResponseWithStructuredOutput(ctx context.Context, msgs []message.Message, tools []tool.BaseTool, info *schema.StructuredOutputInfo) <-chan llm.LLMEvent {
	return m.base.StreamResponseWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *concurrencyTrackingLLM) Model() model.Model {
	return m.base.Model()
}

func (m *concurrencyTrackingLLM) SupportsStructuredOutput() bool {
	return m.base.SupportsStructuredOutput()
}

type toolResultCapturingLLM struct {
	base   *mockLLM
	onCall func(msgs []message.Message)
}

func (m *toolResultCapturingLLM) SendMessages(ctx context.Context, msgs []message.Message, tools []tool.BaseTool) (*llm.LLMResponse, error) {
	if m.onCall != nil {
		m.onCall(msgs)
	}
	return m.base.SendMessages(ctx, msgs, tools)
}

func (m *toolResultCapturingLLM) SendMessagesWithStructuredOutput(ctx context.Context, msgs []message.Message, tools []tool.BaseTool, info *schema.StructuredOutputInfo) (*llm.LLMResponse, error) {
	return m.base.SendMessagesWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *toolResultCapturingLLM) StreamResponse(ctx context.Context, msgs []message.Message, tools []tool.BaseTool) <-chan llm.LLMEvent {
	if m.onCall != nil {
		m.onCall(msgs)
	}
	return m.base.StreamResponse(ctx, msgs, tools)
}

func (m *toolResultCapturingLLM) StreamResponseWithStructuredOutput(ctx context.Context, msgs []message.Message, tools []tool.BaseTool, info *schema.StructuredOutputInfo) <-chan llm.LLMEvent {
	return m.base.StreamResponseWithStructuredOutput(ctx, msgs, tools, info)
}

func (m *toolResultCapturingLLM) Model() model.Model {
	return m.base.Model()
}

func (m *toolResultCapturingLLM) SupportsStructuredOutput() bool {
	return m.base.SupportsStructuredOutput()
}

type echoTool struct{}

func (t *echoTool) Info() tool.ToolInfo {
	return tool.NewToolInfo("echo", "Echoes the input back", struct {
		Text string `json:"text" desc:"Text to echo"`
	}{})
}

func (t *echoTool) Run(_ context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	return tool.NewTextResponse("echo: " + params.Input), nil
}
