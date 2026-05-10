package rag

import (
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/rag"
	"github.com/joakimcarlsson/ai/rag/chunkers/fixed"
)

func TestFixedChunkerProducesChunks(t *testing.T) {
	t.Parallel()

	c := fixed.New(64, 8)
	doc := rag.Document{
		ID:      "doc-1",
		Content: strings.Repeat("alpha bravo charlie delta echo foxtrot golf hotel ", 30),
	}
	chunks := c.Chunk(doc)
	if len(chunks) == 0 {
		t.Fatalf("expected chunks, got 0")
	}

	for i, ch := range chunks {
		if ch.DocumentID != "doc-1" {
			t.Errorf("chunk %d: DocumentID = %q", i, ch.DocumentID)
		}
		if ch.Index != i {
			t.Errorf("chunk %d: Index = %d", i, ch.Index)
		}
		if ch.Content == "" {
			t.Errorf("chunk %d: empty Content", i)
		}
	}

	// Reassembling is approximate (BPE may decode with extra whitespace
	// at boundaries), so just confirm a representative substring of the
	// source survives in at least one chunk.
	found := false
	for _, ch := range chunks {
		if strings.Contains(ch.Content, "alpha bravo charlie") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'alpha bravo charlie' to appear in at least one chunk")
	}
}

func TestFixedChunkerEmptyDoc(t *testing.T) {
	t.Parallel()

	if got := fixed.Default.Chunk(rag.Document{ID: "d", Content: ""}); got != nil {
		t.Errorf("expected nil for empty content, got %d chunks", len(got))
	}
}

func TestFixedChunkerDefaultsClampedSensibly(t *testing.T) {
	t.Parallel()

	doc := rag.Document{ID: "d", Content: strings.Repeat("alpha ", 200)}

	for _, tc := range []struct {
		name          string
		size, overlap int
		wantNonEmpty  bool
		wantOnlyOneCh bool
	}{
		{"normal", 32, 4, true, false},
		{"zero size falls back to default", 0, 0, true, false},
		{"overlap >= size disables overlap", 16, 16, true, false},
		{"negative overlap disables overlap", 16, -5, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chunks := fixed.New(tc.size, tc.overlap).Chunk(doc)
			if tc.wantNonEmpty && len(chunks) == 0 {
				t.Errorf("expected non-empty chunks")
			}
			if tc.wantOnlyOneCh && len(chunks) != 1 {
				t.Errorf("expected 1 chunk, got %d", len(chunks))
			}
		})
	}
}
