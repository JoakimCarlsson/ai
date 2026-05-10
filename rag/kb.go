package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/embeddings"
)

// IDGenerator returns a fresh chunk ID. Used by KnowledgeBase to
// populate Chunk.ID when the configured Chunker leaves it blank.
type IDGenerator func() string

// Option configures a KnowledgeBase. Pass options to New.
type Option func(*config)

type config struct {
	chunker      Chunker
	minScore     float64
	maxDistance  float64
	haveDistance bool
	idGenerator  IDGenerator
}

// WithChunker sets the Chunker used by Ingest. If unset, a built-in
// rune-window chunker (size=2048 runes, overlap=256 runes) is used,
// which is good enough for prototypes. For token-precise chunking
// import rag/chunkers/fixed and pass fixed.Default or fixed.New.
func WithChunker(c Chunker) Option {
	return func(cfg *config) {
		cfg.chunker = c
	}
}

// WithMinScore drops hits whose similarity score is below f at
// retrieval time. Cosine similarity sits in [-1, 1]; typical
// thresholds run between 0.2 and 0.5 depending on the corpus.
func WithMinScore(f float64) Option {
	return func(cfg *config) {
		cfg.minScore = f
	}
}

// WithMaxDistance drops hits whose distance (1 - score) exceeds d at
// retrieval time. Provided for parity with ConvAI-style threshold
// configurations. WithMinScore and WithMaxDistance can both be set;
// hits must satisfy whichever predicates are configured.
func WithMaxDistance(d float64) Option {
	return func(cfg *config) {
		cfg.maxDistance = d
		cfg.haveDistance = true
	}
}

// WithIDGenerator overrides the default chunk ID generator (UUIDv4).
func WithIDGenerator(g IDGenerator) Option {
	return func(cfg *config) {
		if g != nil {
			cfg.idGenerator = g
		}
	}
}

// New constructs a KnowledgeBase. The id scopes ingested chunks
// inside the Store. The embedder is used to embed both chunks (during
// Ingest, with input type "document") and queries (during Retrieve,
// with input type "query"); embedding-aware vendors (e.g., Voyage,
// Cohere) use the input type to score asymmetric retrieval better.
func New(
	id string,
	embedder embeddings.Embedding,
	store Store,
	opts ...Option,
) KnowledgeBase {
	cfg := &config{
		chunker:     defaultChunker(),
		idGenerator: uuid.NewString,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &knowledgeBase{
		id:       id,
		embedder: embedder,
		store:    store,
		cfg:      cfg,
	}
}

type knowledgeBase struct {
	id       string
	embedder embeddings.Embedding
	store    Store
	cfg      *config
}

func (k *knowledgeBase) ID() string { return k.id }

func (k *knowledgeBase) Ingest(
	ctx context.Context,
	docs []Document,
) error {
	if len(docs) == 0 {
		return nil
	}

	var allChunks []Chunk
	for _, doc := range docs {
		for _, chunk := range k.cfg.chunker.Chunk(doc) {
			if chunk.ID == "" {
				chunk.ID = k.cfg.idGenerator()
			}
			allChunks = append(allChunks, chunk)
		}
	}
	if len(allChunks) == 0 {
		return nil
	}

	texts := make([]string, len(allChunks))
	for i, c := range allChunks {
		texts[i] = c.Content
	}

	resp, err := k.embedder.GenerateEmbeddings(ctx, texts, "document")
	if err != nil {
		return fmt.Errorf("rag: embed chunks: %w", err)
	}
	if len(resp.Embeddings) != len(allChunks) {
		return fmt.Errorf(
			"rag: embed chunks: expected %d vectors, got %d",
			len(allChunks),
			len(resp.Embeddings),
		)
	}

	modelID := resp.Model
	if modelID == "" {
		modelID = k.embedder.Model().APIModel
	}

	embedded := make([]EmbeddedChunk, len(allChunks))
	for i, c := range allChunks {
		embedded[i] = EmbeddedChunk{
			Chunk:     c,
			Embedding: resp.Embeddings[i],
			Model:     modelID,
		}
	}

	if err := k.store.Upsert(ctx, k.id, embedded); err != nil {
		return fmt.Errorf("rag: upsert chunks: %w", err)
	}
	return nil
}

func (k *knowledgeBase) Retrieve(
	ctx context.Context,
	query string,
	topK int,
) ([]Hit, error) {
	if strings.TrimSpace(query) == "" || topK <= 0 {
		return nil, nil
	}

	resp, err := k.embedder.GenerateEmbeddings(
		ctx, []string{query}, "query",
	)
	if err != nil {
		return nil, fmt.Errorf("rag: embed query: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, nil
	}

	hits, err := k.store.Search(ctx, k.id, resp.Embeddings[0], topK)
	if err != nil {
		return nil, fmt.Errorf("rag: search store: %w", err)
	}

	return k.filter(hits), nil
}

func (k *knowledgeBase) filter(hits []Hit) []Hit {
	if k.cfg.minScore == 0 && !k.cfg.haveDistance {
		return hits
	}

	out := hits[:0]
	for _, h := range hits {
		if k.cfg.minScore > 0 && h.Score < k.cfg.minScore {
			continue
		}
		if k.cfg.haveDistance && (1.0-h.Score) > k.cfg.maxDistance {
			continue
		}
		out = append(out, h)
	}
	return out
}

// defaultChunker is the built-in rune-window chunker used when
// WithChunker is not set. Size=2048 runes, overlap=256 runes,
// approximating ~512 tokens / ~64 token overlap for English prose.
// Use rag/chunkers/fixed for BPE-token-precise chunking.
func defaultChunker() Chunker {
	return &runeChunker{size: 2048, overlap: 256}
}

type runeChunker struct {
	size    int
	overlap int
}

func (r *runeChunker) Chunk(doc Document) []Chunk {
	runes := []rune(doc.Content)
	if len(runes) == 0 {
		return nil
	}

	step := r.size - r.overlap
	if step <= 0 {
		step = r.size
	}

	var (
		chunks []Chunk
		idx    int
	)
	for start := 0; start < len(runes); start += step {
		end := min(start+r.size, len(runes))
		chunks = append(chunks, Chunk{
			DocumentID: doc.ID,
			Content:    string(runes[start:end]),
			Index:      idx,
			Metadata:   doc.Metadata,
		})
		idx++
		if end == len(runes) {
			break
		}
	}
	return chunks
}
