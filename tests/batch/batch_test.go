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

func newConcurrentProc(
	opts ...batch.Option,
) batch.Processor {
	proc, _ := batch.New("concurrent-test", opts...)
	return proc
}

func TestConcurrentProcess(t *testing.T) {
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
			ID:       "a",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("hello")},
		},
		{
			ID:       "b",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("world")},
		},
		{
			ID:       "c",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("foo")},
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

func TestConcurrentProcessPerItemErrors(t *testing.T) {
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
			ID:       "a",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("1")},
		},
		{
			ID:       "b",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("2")},
		},
		{
			ID:       "c",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("3")},
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

func TestConcurrentProcessEmbeddings(t *testing.T) {
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
	}
	if len(resp.Results[0].EmbedResponse.Embeddings) != 2 {
		t.Errorf(
			"expected 2 embeddings, got %d",
			len(resp.Results[0].EmbedResponse.Embeddings),
		)
	}
}

func TestConcurrentProcessNoClient(t *testing.T) {
	proc := newConcurrentProc()

	requests := []batch.Request{
		{
			ID:       "a",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("hi")},
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

func TestConcurrentProcessEmptyRequests(t *testing.T) {
	proc := newConcurrentProc()
	resp, err := proc.Process(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

func TestConcurrentProcessAutoID(t *testing.T) {
	mock := &mockLLM{
		responses: []mockLLMResponse{{Content: "ok"}},
	}
	proc := newConcurrentProc(batch.WithLLM(mock))

	requests := []batch.Request{
		{
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("hi")},
		},
	}

	resp, err := proc.Process(context.Background(), requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Results[0].ID != "req_0" {
		t.Errorf(
			"expected auto-generated ID 'req_0', got %q",
			resp.Results[0].ID,
		)
	}
}

func TestConcurrentProcessConcurrencyLimit(t *testing.T) {
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

	concurrencyLLM := &concurrencyTrackingLLM{
		base:              mock,
		maxConcurrent:     &maxConcurrent,
		currentConcurrent: &currentConcurrent,
		delay:             20 * time.Millisecond,
	}

	proc := newConcurrentProc(
		batch.WithLLM(concurrencyLLM),
		batch.WithMaxConcurrency(3),
	)

	requests := make([]batch.Request, 10)
	for i := range requests {
		requests[i] = batch.Request{
			ID:       fmt.Sprintf("req_%d", i),
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("test")},
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

func TestProcessAsync(t *testing.T) {
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
			ID:       "a",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("hi")},
		},
		{
			ID:       "b",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("bye")},
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
	for _, ev := range events {
		switch ev.Type {
		case batch.EventComplete:
			hasComplete = true
		case batch.EventItem:
			itemCount++
		}
	}

	if !hasComplete {
		t.Error("expected a complete event")
	}
	if itemCount != 2 {
		t.Errorf("expected 2 item events, got %d", itemCount)
	}
}

func TestProcessAsyncEmpty(t *testing.T) {
	proc := newConcurrentProc()
	ch, err := proc.ProcessAsync(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := make([]batch.Event, 0)
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 1 || events[0].Type != batch.EventComplete {
		t.Error("expected single complete event for empty requests")
	}
}

func TestProgressCallback(t *testing.T) {
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
			ID:       "a",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("1")},
		},
		{
			ID:       "b",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("2")},
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
}

func TestConcurrentProcessContextCancellation(t *testing.T) {
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
			ID:       "a",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("1")},
		},
		{
			ID:       "b",
			Type:     batch.RequestTypeChat,
			Messages: []message.Message{message.NewUserMessage("2")},
		},
	}

	resp, err := proc.Process(ctx, requests)
	if err != nil {
		t.Fatalf("unexpected batch-level error: %v", err)
	}

	if resp.Failed == 0 {
		t.Error(
			"expected at least one failure due to context cancellation",
		)
	}
}

func TestNativeProviderSelection(t *testing.T) {
	proc, err := batch.New(
		model.ProviderOpenAI,
		batch.WithAPIKey("test-key"),
		batch.WithModel(model.OpenAIModels[model.GPT4o]),
	)
	if err != nil {
		t.Fatalf("unexpected error creating OpenAI batch: %v", err)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor for OpenAI")
	}

	proc, err = batch.New(
		model.ProviderAnthropic,
		batch.WithAPIKey("test-key"),
		batch.WithModel(
			model.AnthropicModels[model.Claude4Sonnet],
		),
	)
	if err != nil {
		t.Fatalf(
			"unexpected error creating Anthropic batch: %v",
			err,
		)
	}
	if proc == nil {
		t.Fatal("expected non-nil processor for Anthropic")
	}
}
