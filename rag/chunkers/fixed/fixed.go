// Package fixed provides a token-aware fixed-size chunker for use
// with the rag package. Chunks are sliced by token count using the
// repo-internal BPE tokenizer (tokens.NewCounter), so a chunk of size
// N is approximately N tokens regardless of input language or symbol
// density.
//
// Example:
//
//	import (
//	    "github.com/joakimcarlsson/ai/rag"
//	    "github.com/joakimcarlsson/ai/rag/chunkers/fixed"
//	)
//
//	kb := rag.New("docs", embedder, store,
//	    rag.WithChunker(fixed.Default),
//	)
package fixed

import (
	"github.com/joakimcarlsson/ai/rag"
	"github.com/joakimcarlsson/ai/tokens"
)

// DefaultSize is the default chunk size in tokens (512).
const DefaultSize = 512

// DefaultOverlap is the default overlap between adjacent chunks
// (64 tokens, ~12.5% of DefaultSize).
const DefaultOverlap = 64

// Default is a ready-to-use chunker with size=DefaultSize and
// overlap=DefaultOverlap. Suitable for most prose and markdown
// corpora; tune size/overlap for code, dense tables, or short Q&A.
var Default rag.Chunker = New(DefaultSize, DefaultOverlap)

// New returns a Chunker that splits documents into windows of size
// tokens with the given overlap. Both values are clamped to sensible
// bounds: size <= 0 falls back to DefaultSize, overlap < 0 or >= size
// falls back to 0 (no overlap).
//
// Token counting uses tokens.NewBPETokenizer; the chunker falls back
// to rune-window slicing if the tokenizer cannot be initialised, so
// it never returns an error.
func New(size, overlap int) rag.Chunker {
	if size <= 0 {
		size = DefaultSize
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}
	return &chunker{size: size, overlap: overlap}
}

type chunker struct {
	size    int
	overlap int
}

func (c *chunker) Chunk(doc rag.Document) []rag.Chunk {
	if doc.Content == "" {
		return nil
	}

	tok, err := tokens.NewBPETokenizer()
	if err != nil {
		return runeFallback(doc, c.size, c.overlap)
	}

	ids := tok.Encode(doc.Content)
	if len(ids) == 0 {
		return nil
	}

	step := c.size - c.overlap
	if step <= 0 {
		step = c.size
	}

	var (
		chunks []rag.Chunk
		idx    int
	)
	for start := 0; start < len(ids); start += step {
		end := min(start+c.size, len(ids))
		text := tok.Decode(ids[start:end])
		if text == "" {
			if end == len(ids) {
				break
			}
			continue
		}
		chunks = append(chunks, rag.Chunk{
			DocumentID: doc.ID,
			Content:    text,
			Index:      idx,
			Metadata:   doc.Metadata,
		})
		idx++
		if end == len(ids) {
			break
		}
	}
	return chunks
}

// runeFallback splits on rune count when the BPE tokenizer cannot be
// constructed. Treats size/overlap as rune counts; the result is a
// rough approximation but keeps Chunk a total function.
func runeFallback(doc rag.Document, size, overlap int) []rag.Chunk {
	runes := []rune(doc.Content)
	if len(runes) == 0 {
		return nil
	}

	step := size - overlap
	if step <= 0 {
		step = size
	}

	var (
		chunks []rag.Chunk
		idx    int
	)
	for start := 0; start < len(runes); start += step {
		end := min(start+size, len(runes))
		chunks = append(chunks, rag.Chunk{
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
