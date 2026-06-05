package rag

import (
	"context"
	"hash/fnv"
	"sync"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

// fakeEmbedder is a deterministic, in-memory embedder for tests. It
// produces 32-dim vectors derived from a hash of the input text so
// identical inputs always map to identical vectors and similar inputs
// (sharing tokens) produce vectors with measurable cosine similarity.
type fakeEmbedder struct {
	mu       sync.Mutex
	apiModel string
	calls    [][]string
	dims     int
}

func newFakeEmbedder() *fakeEmbedder {
	return &fakeEmbedder{apiModel: "fake-embed-001", dims: 32}
}

func (e *fakeEmbedder) GenerateEmbeddings(
	_ context.Context,
	texts []string,
	_ ...string,
) (*embeddings.EmbeddingResponse, error) {
	e.mu.Lock()
	e.calls = append(e.calls, append([]string(nil), texts...))
	e.mu.Unlock()

	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = embed(t, e.dims)
	}
	return &embeddings.EmbeddingResponse{
		Embeddings: out,
		Model:      e.apiModel,
	}, nil
}

func (e *fakeEmbedder) GenerateMultimodalEmbeddings(
	_ context.Context,
	_ []embeddings.MultimodalInput,
	_ ...string,
) (*embeddings.EmbeddingResponse, error) {
	return &embeddings.EmbeddingResponse{}, nil
}

func (e *fakeEmbedder) GenerateContextualizedEmbeddings(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return &embeddings.ContextualizedEmbeddingResponse{}, nil
}

func (e *fakeEmbedder) Model() model.EmbeddingModel {
	return model.EmbeddingModel{
		APIModel:      e.apiModel,
		EmbeddingDims: e.dims,
	}
}

// embed produces a stable, normalised vector that gives high cosine
// similarity to texts that share tokens with the input. Tokenises on
// whitespace and assigns one bucket per token via FNV-1a; the
// resulting vector is deterministic given the same text and dims.
func embed(text string, dims int) []float32 {
	v := make([]float32, dims)
	for _, tok := range tokenize(text) {
		h := fnv.New32a()
		_, _ = h.Write([]byte(tok))
		bucket := int(h.Sum32()) % dims
		if bucket < 0 {
			bucket += dims
		}
		v[bucket]++
	}
	return v
}

func tokenize(text string) []string {
	var out []string
	cur := make([]byte, 0, 16)
	for i := range text {
		c := text[i]
		switch {
		case c >= 'A' && c <= 'Z':
			cur = append(cur, c+32)
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			cur = append(cur, c)
		default:
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = cur[:0]
			}
		}
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	return out
}
