package memory

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"testing"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
)

type memMockLLM struct {
	content string
	err     error
}

func (m *memMockLLM) SendMessages(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) (*llm.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.Response{Content: m.content}, nil
}

func (m *memMockLLM) SendMessagesWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	return nil, nil
}

func (m *memMockLLM) StreamResponse(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *memMockLLM) StreamResponseWithStructuredOutput(
	_ context.Context,
	_ []message.Message,
	_ []tool.BaseTool,
	_ *schema.StructuredOutputInfo,
) <-chan llm.Event {
	ch := make(chan llm.Event)
	close(ch)
	return ch
}

func (m *memMockLLM) Model() model.Model {
	return model.Model{ID: "mock", Provider: "mock"}
}

func (m *memMockLLM) SupportsStructuredOutput() bool { return false }

const vecDim = 8

type mockEmbedder struct{}

func (e *mockEmbedder) GenerateEmbeddings(
	_ context.Context,
	texts []string,
	_ ...string,
) (*embeddings.EmbeddingResponse, error) {
	vecs := make([][]float32, len(texts))
	for i, t := range texts {
		vecs[i] = hashToVector(t)
	}
	return &embeddings.EmbeddingResponse{Embeddings: vecs}, nil
}

func (e *mockEmbedder) GenerateMultimodalEmbeddings(
	_ context.Context,
	_ []embeddings.MultimodalInput,
	_ ...string,
) (*embeddings.EmbeddingResponse, error) {
	return &embeddings.EmbeddingResponse{}, nil
}

func (e *mockEmbedder) GenerateContextualizedEmbeddings(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return &embeddings.ContextualizedEmbeddingResponse{}, nil
}

func (e *mockEmbedder) Model() model.EmbeddingModel {
	return model.EmbeddingModel{APIModel: "mock-embed"}
}

func hashToVector(s string) []float32 {
	h := fnv.New64a()
	h.Write([]byte(s))
	seed := h.Sum64()

	vec := make([]float32, vecDim)
	for i := range vec {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		vec[i] = float32(seed%1000) / 1000.0
	}

	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}
	return vec
}

// --- cosineSimilarity tests ---

func TestCosineSimilarity_Identical(t *testing.T) {
	v := []float32{1, 2, 3}
	score := cosineSimilarity(v, v)
	if math.Abs(score-1.0) > 1e-6 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	score := cosineSimilarity(a, b)
	if math.Abs(score) > 1e-6 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{-1, -2, -3}
	score := cosineSimilarity(a, b)
	if math.Abs(score-(-1.0)) > 1e-6 {
		t.Errorf("expected -1.0, got %f", score)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	score := cosineSimilarity(a, b)
	if score != 0 {
		t.Errorf("expected 0 for mismatched lengths, got %f", score)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	score := cosineSimilarity(a, b)
	if score != 0 {
		t.Errorf("expected 0 for zero vector, got %f", score)
	}
}

// --- ExtractFacts tests ---

func TestExtractFacts_KnownInput(t *testing.T) {
	mock := &memMockLLM{
		content: `{"facts":["Name is John","Is a software engineer"]}`,
	}
	msgs := []message.Message{
		message.NewUserMessage("My name is John. I am a software engineer."),
	}

	facts, err := ExtractFacts(context.Background(), mock, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}
	if facts[0] != "Name is John" {
		t.Errorf("expected 'Name is John', got %q", facts[0])
	}
	if facts[1] != "Is a software engineer" {
		t.Errorf("expected 'Is a software engineer', got %q", facts[1])
	}
}

func TestExtractFacts_EmptyConversation(t *testing.T) {
	mock := &memMockLLM{content: "should not be called"}
	facts, err := ExtractFacts(context.Background(), mock, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if facts != nil {
		t.Errorf("expected nil for empty conversation, got %v", facts)
	}
}

func TestExtractFacts_StripsFencing(t *testing.T) {
	mock := &memMockLLM{content: "```json\n{\"facts\":[\"Likes pizza\"]}\n```"}
	msgs := []message.Message{
		message.NewUserMessage("I like pizza"),
	}

	facts, err := ExtractFacts(context.Background(), mock, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(facts) != 1 || facts[0] != "Likes pizza" {
		t.Errorf("expected [Likes pizza], got %v", facts)
	}
}

func TestExtractFacts_LLMError(t *testing.T) {
	mock := &memMockLLM{err: fmt.Errorf("connection refused")}
	msgs := []message.Message{
		message.NewUserMessage("hello"),
	}

	_, err := ExtractFacts(context.Background(), mock, msgs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Deduplicate tests ---

func TestDeduplicate_NoExisting(t *testing.T) {
	mock := &memMockLLM{}
	result, err := Deduplicate(context.Background(), mock, "new fact", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(result.Decisions))
	}
	if result.Decisions[0].Event != DedupEventAdd {
		t.Errorf("expected ADD, got %s", result.Decisions[0].Event)
	}
	if result.Decisions[0].Text != "new fact" {
		t.Errorf("expected 'new fact', got %q", result.Decisions[0].Text)
	}
}

func TestDeduplicate_UpdateExisting(t *testing.T) {
	mock := &memMockLLM{
		content: `{"decisions":[{"event":"UPDATE","memory_id":"mem-1","text":"Name is John Doe (updated)"}]}`,
	}
	existing := []Entry{{ID: "mem-1", Content: "Name is John"}}

	result, err := Deduplicate(
		context.Background(),
		mock,
		"Name is John Doe",
		existing,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(result.Decisions))
	}
	d := result.Decisions[0]
	if d.Event != DedupEventUpdate {
		t.Errorf("expected UPDATE, got %s", d.Event)
	}
	if d.MemoryID != "mem-1" {
		t.Errorf("expected memory_id 'mem-1', got %q", d.MemoryID)
	}
}

func TestDeduplicate_DeleteExisting(t *testing.T) {
	mock := &memMockLLM{
		content: `{"decisions":[{"event":"DELETE","memory_id":"mem-1","text":"old fact"}]}`,
	}
	existing := []Entry{{ID: "mem-1", Content: "old fact"}}

	result, err := Deduplicate(
		context.Background(),
		mock,
		"contradicting fact",
		existing,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decisions[0].Event != DedupEventDelete {
		t.Errorf("expected DELETE, got %s", result.Decisions[0].Event)
	}
}

func TestDeduplicate_NoneEvent(t *testing.T) {
	mock := &memMockLLM{
		content: `{"decisions":[{"event":"NONE","text":"already known"}]}`,
	}
	existing := []Entry{{ID: "mem-1", Content: "already known"}}

	result, err := Deduplicate(
		context.Background(),
		mock,
		"already known",
		existing,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decisions[0].Event != DedupEventNone {
		t.Errorf("expected NONE, got %s", result.Decisions[0].Event)
	}
}

func TestDeduplicate_JSONParseFail(t *testing.T) {
	mock := &memMockLLM{content: "this is not json at all"}
	existing := []Entry{{ID: "mem-1", Content: "some fact"}}

	result, err := Deduplicate(context.Background(), mock, "new fact", existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Decisions) != 1 {
		t.Fatalf("expected 1 fallback decision, got %d", len(result.Decisions))
	}
	if result.Decisions[0].Event != DedupEventAdd {
		t.Errorf("expected fallback ADD, got %s", result.Decisions[0].Event)
	}
	if result.Decisions[0].Text != "new fact" {
		t.Errorf(
			"expected fallback text 'new fact', got %q",
			result.Decisions[0].Text,
		)
	}
}

// --- NewStore (in-memory) tests ---

func TestMemoryStore_StoreAndSearch(t *testing.T) {
	embed := &mockEmbedder{}
	counter := 0
	store := NewStore(embed, WithIDGenerator(func() string {
		counter++
		return fmt.Sprintf("id-%d", counter)
	}))

	ctx := context.Background()
	if err := store.Store(ctx, "user-1", "I like Go programming", nil); err != nil {
		t.Fatalf("store failed: %v", err)
	}
	if err := store.Store(ctx, "user-1", "My favorite color is blue", nil); err != nil {
		t.Fatalf("store failed: %v", err)
	}
	if err := store.Store(ctx, "user-1", "I enjoy Go concurrency patterns", nil); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	results, err := store.Search(ctx, "user-1", "I like Go programming", 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Content != "I like Go programming" {
		t.Errorf(
			"expected top result to be exact match, got %q",
			results[0].Content,
		)
	}
}

func TestMemoryStore_GetAll(t *testing.T) {
	embed := &mockEmbedder{}
	counter := 0
	store := NewStore(embed, WithIDGenerator(func() string {
		counter++
		return fmt.Sprintf("id-%d", counter)
	}))

	ctx := context.Background()
	store.Store(ctx, "user-1", "fact A", nil)
	store.Store(ctx, "user-1", "fact B", nil)
	store.Store(ctx, "user-1", "fact C", nil)

	results, err := store.GetAll(ctx, "user-1", 10)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 entries, got %d", len(results))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	embed := &mockEmbedder{}
	counter := 0
	store := NewStore(embed, WithIDGenerator(func() string {
		counter++
		return fmt.Sprintf("id-%d", counter)
	}))

	ctx := context.Background()
	store.Store(ctx, "user-1", "to be deleted", nil)
	store.Store(ctx, "user-1", "to keep", nil)

	if err := store.Delete(ctx, "id-1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	results, err := store.GetAll(ctx, "user-1", 10)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 entry after delete, got %d", len(results))
	}
	if results[0].Content != "to keep" {
		t.Errorf("wrong entry remaining: %q", results[0].Content)
	}
}

func TestMemoryStore_Update(t *testing.T) {
	embed := &mockEmbedder{}
	counter := 0
	store := NewStore(embed, WithIDGenerator(func() string {
		counter++
		return fmt.Sprintf("id-%d", counter)
	}))

	ctx := context.Background()
	store.Store(ctx, "user-1", "Name is John", nil)

	if err := store.Update(ctx, "id-1", "Name is John Doe", nil); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	results, err := store.GetAll(ctx, "user-1", 10)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(results))
	}
	if results[0].Content != "Name is John Doe" {
		t.Errorf(
			"expected updated content 'Name is John Doe', got %q",
			results[0].Content,
		)
	}
}

// --- FileStore tests ---

func TestFileStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	embed := &mockEmbedder{}
	counter := 0
	idGen := WithIDGenerator(func() string {
		counter++
		return fmt.Sprintf("fid-%d", counter)
	})

	store := FileStore(dir, embed, idGen)
	ctx := context.Background()

	store.Store(ctx, "owner-1", "fact alpha", nil)
	store.Store(ctx, "owner-1", "fact beta", nil)

	store2 := FileStore(dir, embed, idGen)
	results, err := store2.GetAll(ctx, "owner-1", 10)
	if err != nil {
		t.Fatalf("GetAll on new store failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 entries after round-trip, got %d", len(results))
	}
}

func TestFileStore_SearchRanking(t *testing.T) {
	dir := t.TempDir()
	embed := &mockEmbedder{}
	counter := 0
	store := FileStore(dir, embed, WithIDGenerator(func() string {
		counter++
		return fmt.Sprintf("fid-%d", counter)
	}))

	ctx := context.Background()
	store.Store(ctx, "owner-1", "I like Go programming", nil)
	store.Store(ctx, "owner-1", "My cat is named Whiskers", nil)
	store.Store(ctx, "owner-1", "I enjoy Go concurrency", nil)

	results, err := store.Search(ctx, "owner-1", "I like Go programming", 3)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) < 1 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Content != "I like Go programming" {
		t.Errorf(
			"expected top result to be exact query match, got %q",
			results[0].Content,
		)
	}
}
