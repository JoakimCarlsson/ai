package batch

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"google.golang.org/genai"
)

type mockLLM struct {
	mu        sync.Mutex
	responses []mockLLMResponse
	callIndex int
}

type mockLLMResponse struct {
	Content string
	Err     error
	Delay   time.Duration
}

func (m *mockLLM) SendMessages(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) (*llm.Response, error) {
	m.mu.Lock()
	resp := m.responses[m.callIndex%len(m.responses)]
	m.callIndex++
	m.mu.Unlock()

	if resp.Delay > 0 {
		time.Sleep(resp.Delay)
	}
	if resp.Err != nil {
		return nil, resp.Err
	}
	return &llm.Response{
		Content:      resp.Content,
		FinishReason: message.FinishReasonEndTurn,
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
	close(ch)
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
	return model.Model{ID: "mock"}
}

func (m *mockLLM) SupportsStructuredOutput() bool {
	return false
}

type mockEmbedding struct {
	response *embeddings.EmbeddingResponse
	err      error
}

func (m *mockEmbedding) GenerateEmbeddings(
	_ context.Context,
	_ []string,
	_ ...string,
) (*embeddings.EmbeddingResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockEmbedding) GenerateMultimodalEmbeddings(
	_ context.Context,
	_ []embeddings.MultimodalInput,
	_ ...string,
) (*embeddings.EmbeddingResponse, error) {
	return nil, nil
}

func (m *mockEmbedding) GenerateContextualizedEmbeddings(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return nil, nil
}

func (m *mockEmbedding) Model() model.EmbeddingModel {
	return model.EmbeddingModel{ID: "mock-embed"}
}

type concurrencyTrackingLLM struct {
	base              *mockLLM
	maxConcurrent     *atomic.Int32
	currentConcurrent *atomic.Int32
	delay             time.Duration
}

func (m *concurrencyTrackingLLM) SendMessages(
	ctx context.Context,
	msgs []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
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

func (m *concurrencyTrackingLLM) SendMessagesWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *concurrencyTrackingLLM) StreamResponse(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *concurrencyTrackingLLM) StreamResponseWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *concurrencyTrackingLLM) Model() model.Model {
	return model.Model{ID: "mock"}
}

func (m *concurrencyTrackingLLM) SupportsStructuredOutput() bool {
	return false
}

func newConcurrentProc(
	opts ...batch.Option,
) batch.Processor {
	proc, _ := batch.New("concurrent-test", opts...)
	return proc
}

func TestNew_ProviderRouting(t *testing.T) {
	tests := []struct {
		name     string
		provider model.Provider
	}{
		{"OpenAI", model.ProviderOpenAI},
		{"Anthropic", model.ProviderAnthropic},
		{"Gemini", model.ProviderGemini},
		{"Unknown fallback", "unknown-provider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc, err := batch.New(
				tt.provider,
				batch.WithAPIKey("test-key"),
				batch.WithModel(model.Model{APIModel: "test"}),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if proc == nil {
				t.Fatal("expected non-nil processor")
			}
		})
	}
}

func TestNew_AllOptions(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{{Content: "ok"}},
	}
	mockEmbed := &mockEmbedding{
		response: &embeddings.EmbeddingResponse{},
	}

	proc, err := batch.New(
		"test",
		batch.WithAPIKey("key"),
		batch.WithModel(model.Model{APIModel: "m"}),
		batch.WithEmbeddingModel(
			model.EmbeddingModel{APIModel: "e"},
		),
		batch.WithMaxTokens(1000),
		batch.WithMaxConcurrency(5),
		batch.WithPollInterval(10*time.Second),
		batch.WithTimeout(30*time.Second),
		batch.WithLLM(mock),
		batch.WithEmbedding(mockEmbed),
		batch.WithProgressCallback(func(_ batch.Progress) {}),
		batch.WithOpenAIOptions(),
		batch.WithAnthropicOptions(),
		batch.WithGeminiOptions(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestProcess_ChatCompletions(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{
			{Content: "response 1"},
			{Content: "response 2"},
			{Content: "response 3"},
		},
	}

	proc := newConcurrentProc(
		batch.WithLLM(mock),
		batch.WithMaxConcurrency(2),
	)

	requests := []batch.Request{
		{
			ID:   "a",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("hello"),
			},
		},
		{
			ID:   "b",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("world"),
			},
		},
		{
			ID:   "c",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("foo"),
			},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Total)
	}
	if resp.Completed != 3 {
		t.Errorf("expected completed 3, got %d", resp.Completed)
	}
	if resp.Failed != 0 {
		t.Errorf("expected failed 0, got %d", resp.Failed)
	}

	for _, r := range resp.Results {
		if r.Err != nil {
			t.Errorf("unexpected error for %s: %v", r.ID, r.Err)
		}
		if r.ChatResponse == nil {
			t.Errorf("expected chat response for %s", r.ID)
		}
	}
}

func TestProcess_PerItemErrors(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{
			{Content: "ok"},
			{Err: fmt.Errorf("model overloaded")},
			{Content: "ok too"},
		},
	}

	proc := newConcurrentProc(
		batch.WithLLM(mock),
		batch.WithMaxConcurrency(1),
	)

	requests := []batch.Request{
		{
			ID:   "a",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("1"),
			},
		},
		{
			ID:   "b",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("2"),
			},
		},
		{
			ID:   "c",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("3"),
			},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("batch should not fail: %v", err)
	}

	if resp.Completed != 2 {
		t.Errorf("expected 2 completed, got %d", resp.Completed)
	}
	if resp.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", resp.Failed)
	}

	errorCount := 0
	for _, r := range resp.Results {
		if r.Err != nil {
			errorCount++
		}
	}
	if errorCount != 1 {
		t.Errorf("expected exactly 1 error, got %d", errorCount)
	}
}

func TestProcess_Embeddings(t *testing.T) {
	mock := &mockEmbedding{
		response: &embeddings.EmbeddingResponse{
			Embeddings: [][]float32{{0.1, 0.2}, {0.3, 0.4}},
			Usage:      embeddings.EmbeddingUsage{TotalTokens: 10},
			Model:      "mock",
		},
	}

	proc := newConcurrentProc(batch.WithEmbedding(mock))

	requests := []batch.Request{
		{
			ID:    "e1",
			Type:  batch.RequestTypeEmbedding,
			Texts: []string{"hello", "world"},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", resp.Completed)
	}
	if resp.Results[0].EmbedResponse == nil {
		t.Fatal("expected embedding response")
		return
	}
	if len(resp.Results[0].EmbedResponse.Embeddings) != 2 {
		t.Errorf(
			"expected 2 embeddings, got %d",
			len(resp.Results[0].EmbedResponse.Embeddings),
		)
	}
}

func TestProcess_EmbeddingError(t *testing.T) {
	mock := &mockEmbedding{
		err: fmt.Errorf("embedding service unavailable"),
	}

	proc := newConcurrentProc(batch.WithEmbedding(mock))

	requests := []batch.Request{
		{
			ID:    "e1",
			Type:  batch.RequestTypeEmbedding,
			Texts: []string{"hello"},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", resp.Failed)
	}
	if resp.Results[0].Err == nil {
		t.Error("expected error on result")
	}
}

func TestProcess_MixedChatAndEmbedding(t *testing.T) {
	mockChat := &mockLLM{
		responses: []mockLLMResponse{{Content: "chat response"}},
	}
	mockEmbed := &mockEmbedding{
		response: &embeddings.EmbeddingResponse{
			Embeddings: [][]float32{{0.1}},
			Usage:      embeddings.EmbeddingUsage{TotalTokens: 5},
			Model:      "mock",
		},
	}

	proc := newConcurrentProc(
		batch.WithLLM(mockChat),
		batch.WithEmbedding(mockEmbed),
		batch.WithMaxConcurrency(2),
	)

	requests := []batch.Request{
		{
			ID:   "chat1",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("hi"),
			},
		},
		{
			ID:    "embed1",
			Type:  batch.RequestTypeEmbedding,
			Texts: []string{"test"},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Completed != 2 {
		t.Errorf("expected 2 completed, got %d", resp.Completed)
	}

	chatFound := false
	embedFound := false
	for _, r := range resp.Results {
		if r.ID == "chat1" && r.ChatResponse != nil {
			chatFound = true
		}
		if r.ID == "embed1" && r.EmbedResponse != nil {
			embedFound = true
		}
	}
	if !chatFound {
		t.Error("expected chat result")
	}
	if !embedFound {
		t.Error("expected embedding result")
	}
}

func TestProcess_NoLLMClient(t *testing.T) {
	proc := newConcurrentProc()

	requests := []batch.Request{
		{
			ID:   "a",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("hi"),
			},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", resp.Failed)
	}
	if resp.Results[0].Err != batch.ErrNoLLMClient {
		t.Errorf(
			"expected ErrNoLLMClient, got %v",
			resp.Results[0].Err,
		)
	}
}

func TestProcess_NoEmbeddingClient(t *testing.T) {
	proc := newConcurrentProc()

	requests := []batch.Request{
		{
			ID:    "a",
			Type:  batch.RequestTypeEmbedding,
			Texts: []string{"test"},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Results[0].Err != batch.ErrNoEmbeddingClient {
		t.Errorf(
			"expected ErrNoEmbeddingClient, got %v",
			resp.Results[0].Err,
		)
	}
}

func TestProcess_EmptyRequests(t *testing.T) {
	proc := newConcurrentProc()
	resp, err := proc.Process(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestProcess_AutoGeneratesIDs(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{
			{Content: "ok"},
			{Content: "ok"},
		},
	}
	proc := newConcurrentProc(batch.WithLLM(mock))

	requests := []batch.Request{
		{
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("first"),
			},
		},
		{
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("second"),
			},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Results[0].ID != "req_0" {
		t.Errorf("expected ID 'req_0', got %q", resp.Results[0].ID)
	}
	if resp.Results[1].ID != "req_1" {
		t.Errorf("expected ID 'req_1', got %q", resp.Results[1].ID)
	}
}

func TestProcess_PreservesCustomIDs(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{{Content: "ok"}},
	}
	proc := newConcurrentProc(batch.WithLLM(mock))

	requests := []batch.Request{
		{
			ID:   "my-custom-id",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("hi"),
			},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Results[0].ID != "my-custom-id" {
		t.Errorf(
			"expected ID 'my-custom-id', got %q",
			resp.Results[0].ID,
		)
	}
}

func TestProcess_ConcurrencyLimit(t *testing.T) {
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	mock := &mockLLM{
		responses: make([]mockLLMResponse, 10),
	}
	for i := range mock.responses {
		mock.responses[i] = mockLLMResponse{
			Content: fmt.Sprintf("resp_%d", i),
			Delay:   20 * time.Millisecond,
		}
	}

	tracker := &concurrencyTrackingLLM{
		base:              mock,
		maxConcurrent:     &maxConcurrent,
		currentConcurrent: &currentConcurrent,
		delay:             20 * time.Millisecond,
	}

	proc := newConcurrentProc(
		batch.WithLLM(tracker),
		batch.WithMaxConcurrency(3),
	)

	requests := make([]batch.Request, 10)
	for i := range requests {
		requests[i] = batch.Request{
			ID:   fmt.Sprintf("req_%d", i),
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("test"),
			},
		}
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Completed != 10 {
		t.Errorf("expected 10 completed, got %d", resp.Completed)
	}
	if maxConcurrent.Load() > 3 {
		t.Errorf(
			"expected max concurrency <= 3, got %d",
			maxConcurrent.Load(),
		)
	}
}

func TestProcess_ContextCancellation(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{
			{Content: "ok", Delay: 100 * time.Millisecond},
			{Content: "ok", Delay: 100 * time.Millisecond},
		},
	}

	proc := newConcurrentProc(
		batch.WithLLM(mock),
		batch.WithMaxConcurrency(1),
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Millisecond,
	)
	defer cancel()

	requests := []batch.Request{
		{
			ID:   "a",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("1"),
			},
		},
		{
			ID:   "b",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("2"),
			},
		},
	}

	resp, err := proc.Process(ctx, requests)
	if err != nil {
		t.Fatalf("unexpected batch-level error: %v", err)
	}

	if resp.Failed == 0 {
		t.Error(
			"expected at least one failure due to cancellation",
		)
	}
}

func TestProcessAsync_ChatCompletions(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{
			{Content: "async 1"},
			{Content: "async 2"},
		},
	}

	proc := newConcurrentProc(
		batch.WithLLM(mock),
		batch.WithMaxConcurrency(2),
	)

	requests := []batch.Request{
		{
			ID:   "a",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("hi"),
			},
		},
		{
			ID:   "b",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("bye"),
			},
		},
	}

	ch, err := proc.ProcessAsync(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var events []batch.Event
	for ev := range ch {
		events = append(events, ev)
	}

	hasComplete := false
	itemCount := 0
	progressCount := 0
	for _, ev := range events {
		switch ev.Type {
		case batch.EventComplete:
			hasComplete = true
			if ev.Progress == nil {
				t.Error("complete event should have progress")
			}
		case batch.EventItem:
			itemCount++
			if ev.Result == nil {
				t.Error("item event should have result")
			}
		case batch.EventProgress:
			progressCount++
			if ev.Progress == nil {
				t.Error("progress event should have progress")
			}
		}
	}

	if !hasComplete {
		t.Error("expected a complete event")
	}
	if itemCount != 2 {
		t.Errorf("expected 2 item events, got %d", itemCount)
	}
	if progressCount < 2 {
		t.Errorf(
			"expected at least 2 progress events, got %d",
			progressCount,
		)
	}
}

func TestProcessAsync_EmptyRequests(t *testing.T) {
	proc := newConcurrentProc()
	ch, err := proc.ProcessAsync(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := make([]batch.Event, 0)
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != batch.EventComplete {
		t.Error("expected complete event")
	}
	if events[0].Progress == nil {
		t.Fatal("expected progress on complete event")
		return
	}
	if events[0].Progress.Total != 0 {
		t.Errorf("expected total 0, got %d", events[0].Progress.Total)
	}
}

func TestProcessAsync_AutoGeneratesIDs(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{{Content: "ok"}},
	}
	proc := newConcurrentProc(batch.WithLLM(mock))

	requests := []batch.Request{
		{
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("hi"),
			},
		},
	}

	ch, err := proc.ProcessAsync(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for ev := range ch {
		if ev.Type == batch.EventItem && ev.Result != nil {
			if ev.Result.ID != "req_0" {
				t.Errorf(
					"expected ID 'req_0', got %q",
					ev.Result.ID,
				)
			}
		}
	}
}

func TestProgressCallback_Invoked(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{
			{Content: "r1", Delay: 5 * time.Millisecond},
			{Content: "r2", Delay: 5 * time.Millisecond},
		},
	}

	var mu sync.Mutex
	var updates []batch.Progress

	proc := newConcurrentProc(
		batch.WithLLM(mock),
		batch.WithMaxConcurrency(1),
		batch.WithProgressCallback(func(p batch.Progress) {
			mu.Lock()
			updates = append(updates, p)
			mu.Unlock()
		}),
	)

	requests := []batch.Request{
		{
			ID:   "a",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("1"),
			},
		},
		{
			ID:   "b",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("2"),
			},
		},
	}

	_, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(updates) < 2 {
		t.Errorf(
			"expected at least 2 progress updates, got %d",
			len(updates),
		)
	}
	for _, u := range updates {
		if u.Total != 2 {
			t.Errorf("expected total 2 in progress, got %d", u.Total)
		}
		if u.Status != "processing" {
			t.Errorf(
				"expected status 'processing', got %q",
				u.Status,
			)
		}
	}
}

func TestNativeProviderSelection_OpenAI(t *testing.T) {
	proc, err := batch.New(
		model.ProviderOpenAI,
		batch.WithAPIKey("test-key"),
		batch.WithModel(model.OpenAIModels[model.GPT4o]),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestNativeProviderSelection_Anthropic(t *testing.T) {
	proc, err := batch.New(
		model.ProviderAnthropic,
		batch.WithAPIKey("test-key"),
		batch.WithModel(model.AnthropicModels[model.Claude4Sonnet]),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestNativeProviderSelection_Gemini(t *testing.T) {
	proc, err := batch.New(
		model.ProviderGemini,
		batch.WithAPIKey("test-key"),
		batch.WithModel(model.Model{APIModel: "gemini-2.5-flash"}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestNativeProviderSelection_FallbackToConcurrent(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{{Content: "ok"}},
	}

	proc, err := batch.New(
		model.ProviderGROQ,
		batch.WithLLM(mock),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requests := []batch.Request{
		{
			ID:   "a",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("hi"),
			},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", resp.Completed)
	}
}

func TestOpenAIOptions(t *testing.T) {
	proc, err := batch.New(
		model.ProviderOpenAI,
		batch.WithAPIKey("test-key"),
		batch.WithModel(model.OpenAIModels[model.GPT4o]),
		batch.WithOpenAIOptions(
			batch.WithOpenAIBaseURL("https://custom.api.com/v1"),
			batch.WithOpenAIExtraHeaders(map[string]string{
				"X-Custom": "value",
			}),
		),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestGeminiOptions(t *testing.T) {
	proc, err := batch.New(
		model.ProviderGemini,
		batch.WithAPIKey("test-key"),
		batch.WithModel(model.Model{APIModel: "gemini-2.5-flash"}),
		batch.WithGeminiOptions(
			batch.WithGeminiBackend(genai.BackendVertexAI),
		),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestProcess_AllResultsHaveIndex(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{
			{Content: "a"},
			{Content: "b"},
			{Content: "c"},
		},
	}

	proc := newConcurrentProc(
		batch.WithLLM(mock),
		batch.WithMaxConcurrency(3),
	)

	requests := []batch.Request{
		{
			ID:   "x",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("1"),
			},
		},
		{
			ID:   "y",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("2"),
			},
		},
		{
			ID:   "z",
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("3"),
			},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, r := range resp.Results {
		if r.Index != i {
			t.Errorf(
				"result %d: expected index %d, got %d",
				i, i, r.Index,
			)
		}
	}
}

func TestProcess_LargeRequestCount(t *testing.T) {
	responses := make([]mockLLMResponse, 100)
	for i := range responses {
		responses[i] = mockLLMResponse{
			Content: fmt.Sprintf("resp_%d", i),
		}
	}
	mock := &mockLLM{responses: responses}

	proc := newConcurrentProc(
		batch.WithLLM(mock),
		batch.WithMaxConcurrency(20),
	)

	requests := make([]batch.Request, 100)
	for i := range requests {
		requests[i] = batch.Request{
			ID:   fmt.Sprintf("req_%d", i),
			Type: batch.RequestTypeChat,
			Messages: []message.Message{
				message.NewUserMessage("test"),
			},
		}
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 100 {
		t.Errorf("expected total 100, got %d", resp.Total)
	}
	if resp.Completed != 100 {
		t.Errorf(
			"expected 100 completed, got %d",
			resp.Completed,
		)
	}
	if resp.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", resp.Failed)
	}
}
