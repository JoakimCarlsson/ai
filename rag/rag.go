// Package rag provides primitives for retrieval-augmented generation:
// chunking documents, embedding chunks, storing them in a vector store,
// and retrieving semantically similar chunks for a query.
//
// A KnowledgeBase composes an embedder + chunker + store. Pass it to
// an agent or voice agent via WithKnowledgeBase to auto-inject relevant
// context into LLM calls, or use rag.SearchTool to expose retrieval as
// a tool the LLM can call explicitly.
//
// Example usage:
//
//	import (
//	    "github.com/joakimcarlsson/ai/embeddings/voyage"
//	    "github.com/joakimcarlsson/ai/rag"
//	    ragmem "github.com/joakimcarlsson/ai/rag/store/memory"
//	)
//
//	embedder := voyage.NewEmbedding(voyage.WithAPIKey(key))
//	store    := ragmem.New()
//	kb       := rag.New("docs", embedder, store)
//
//	_ = kb.Ingest(ctx, []rag.Document{{ID: "faq", Content: "..."}})
//	hits, _ := kb.Retrieve(ctx, "What is X?", 5)
package rag

import "context"

// Document is a unit of source content to be ingested into a knowledge
// base. Users construct Documents themselves; loaders for PDF, URL, etc.
// ship as separate sub-modules.
type Document struct {
	ID       string
	Content  string
	Metadata map[string]any
}

// Chunk is a Document split into a retrievable unit.
type Chunk struct {
	ID         string
	DocumentID string
	Content    string
	Index      int
	Metadata   map[string]any
}

// EmbeddedChunk is a Chunk paired with its computed embedding vector.
//
// Model carries the APIModel identifier of the embedding model used to
// produce Embedding. Stored alongside the vector at ingest time so a
// future re-embed pass can detect drift between the index and the
// active embedder.
type EmbeddedChunk struct {
	Chunk
	Embedding []float32
	Model     string
}

// Hit is a Chunk returned from a similarity search with its score.
// Score is in [-1, 1] for cosine-similarity stores; higher is better.
type Hit struct {
	Chunk
	Score float64
}

// Chunker splits a Document into Chunks. Implementations are pure (no
// IO) and deterministic for a given input.
type Chunker interface {
	Chunk(Document) []Chunk
}

// SearchOption tunes a Store.Search call. Implementations are free to
// ignore unrecognised options; v1 ships only the type so future
// additions (filters, score thresholds, namespaces) are non-breaking.
type SearchOption interface {
	applySearchOption(*searchConfig)
}

// searchConfig is the merged effect of all SearchOptions on a single
// Search call. Stores read it instead of each implementation re-doing
// option plumbing. Reserved for future additions (Filter, MinScore,
// Namespace, ...). See ApplySearchOptions for the public consumer
// helper that store implementations call.
type searchConfig struct{}

// ApplySearchOptions reduces a slice of SearchOptions into the merged
// SearchConfig that store implementations consult. Stores typically
// call this once at the top of Search and read fields from the result.
//
// Returns a zero SearchConfig in v1 (no options ship yet); the helper
// exists so the v1 surface is forward-compatible with adding options
// later without changing every store.
func ApplySearchOptions(opts ...SearchOption) SearchConfig {
	var cfg searchConfig
	for _, opt := range opts {
		opt.applySearchOption(&cfg)
	}
	return SearchConfig{cfg: cfg}
}

// SearchConfig is the publicly readable view of a merged SearchOption
// set. It is intentionally opaque in v1; accessor methods land here as
// new options are added.
type SearchConfig struct {
	//nolint:unused // reserved for future option fields
	cfg searchConfig
}

// Store persists pre-embedded chunks under a knowledge-base id and
// serves vector similarity queries over them. Implementations live
// under rag/store/{memory,pgvector,...}.
type Store interface {
	// Upsert inserts or replaces a batch of chunks under kbID. Chunk
	// IDs are unique within a knowledge base; re-inserting the same
	// ID overwrites.
	Upsert(ctx context.Context, kbID string, chunks []EmbeddedChunk) error

	// Search returns up to k chunks under kbID ranked by similarity
	// to embedding. Variadic opts lets implementations grow filters,
	// score thresholds, etc. without breaking the interface;
	// unrecognised options are ignored.
	Search(
		ctx context.Context,
		kbID string,
		embedding []float32,
		k int,
		opts ...SearchOption,
	) ([]Hit, error)

	// Delete removes the chunk with the given ID from any knowledge
	// base it lives under. Returns nil if the chunk does not exist.
	Delete(ctx context.Context, chunkID string) error
}

// KnowledgeBase orchestrates ingest and retrieval for one corpus,
// scoped under an ID. Composes an embedder + chunker + store.
type KnowledgeBase interface {
	// ID returns the knowledge-base identifier passed to New.
	ID() string

	// Ingest chunks each document, embeds the chunks, and upserts
	// them into the configured Store under the KB's ID.
	Ingest(ctx context.Context, docs []Document) error

	// Retrieve embeds the query, searches the Store for the top-k
	// matches, and applies the KB's configured score threshold.
	Retrieve(ctx context.Context, query string, k int) ([]Hit, error)
}
