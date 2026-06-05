// Command rag-pgvector-eval runs an evaluation of the rag pipeline
// (pgvector-backed) against a golden set defined in cases.go. It
// reports retrieval recall@k, MRR, and LLM-judge scores for
// faithfulness and correctness.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/joakimcarlsson/ai/agent"
	embedopenai "github.com/joakimcarlsson/ai/embeddings/openai"
	"github.com/joakimcarlsson/ai/eval"
	"github.com/joakimcarlsson/ai/eval/judge"
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rag"
	"github.com/joakimcarlsson/ai/rag/chunkers/fixed"
	pgstore "github.com/joakimcarlsson/ai/rag/store/pgvector"
)

const (
	dbURL = "postgres://rag:rag@localhost:5433/rag?sslmode=disable"
	kbID  = "support-docs"
	dims  = 1536
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}
	ctx := context.Background()

	if err := pgstore.Migrate(ctx, dbURL, dims); err != nil {
		log.Fatalf("migrate: %v\n(is docker-compose up?)", err)
	}

	embedder := embedopenai.NewEmbedding(
		embedopenai.WithAPIKey(apiKey),
		embedopenai.WithModel(
			model.OpenAIEmbeddingModels[model.TextEmbedding3Small],
		),
	)

	store, err := pgstore.New(ctx, dbURL, dims)
	if err != nil {
		log.Fatalf("pgvector store: %v", err)
	}
	kb := rag.New(kbID, embedder, store, rag.WithChunker(fixed.Default))

	if err := ingestIfNeeded(ctx, kb); err != nil {
		log.Fatalf("ingest: %v", err)
	}

	llmClient := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT54Mini]),
		llmopenai.WithMaxTokens(512),
	)
	judgeClient := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT54Mini]),
		llmopenai.WithMaxTokens(384),
	)

	subject := newRAGSubject(llmClient, kb)

	metrics := []eval.Metric[RAGExpectations, RAGOutput]{
		retrievalHitAt(2),
		retrievalHitAt(5),
		mrr(),
		judge.Faithfulness[RAGExpectations, RAGOutput](
			judgeClient,
			func(o eval.Output[RAGOutput]) string { return o.Extras.RetrievedContext },
			func(c eval.Case[RAGExpectations]) bool { return c.Extras.OffTopic },
		),
		judge.Correctness[RAGExpectations, RAGOutput](
			judgeClient,
			func(c eval.Case[RAGExpectations]) bool { return c.Extras.OffTopic },
		),
	}

	rep := eval.Run(ctx, subject, goldenCases, metrics)
	fmt.Print(eval.FormatReport(rep))
}

// ragSubject implements eval.Subject[RAGOutput]. It runs the user
// query through an agent.Agent with a knowledge base wired in, and
// records what the agent retrieved into Output.Extras for the
// metrics layer to score.
type ragSubject struct {
	agent *agent.Agent
	tap   *retrievalTap
}

func (r *ragSubject) SubjectName() string { return "rag-pgvector + gpt-5-4-mini" }

func newRAGSubject(llmClient llm.LLM, kb rag.KnowledgeBase) *ragSubject {
	tap := &retrievalTap{inner: kb}
	a := agent.New(llmClient,
		agent.WithSystemPrompt(
			"You answer customer-support questions using the supplied knowledge base. "+
				"Cite the source document ID in square brackets after each claim. "+
				"If the answer is not in the knowledge base, decline plainly and do not guess.",
		),
		agent.WithKnowledgeBase(tap),
		agent.WithTools(rag.SearchTool(tap)),
	)
	return &ragSubject{agent: a, tap: tap}
}

func (r *ragSubject) Run(
	ctx context.Context,
	input string,
) (eval.Output[RAGOutput], error) {
	r.tap.reset()
	resp, err := r.agent.Chat(ctx, input)
	if err != nil {
		return eval.Output[RAGOutput]{}, err
	}

	hits := r.tap.allHits()
	docIDs := make([]string, 0, len(hits))
	seen := map[string]struct{}{}
	for _, h := range hits {
		if _, ok := seen[h.DocumentID]; ok {
			continue
		}
		seen[h.DocumentID] = struct{}{}
		docIDs = append(docIDs, h.DocumentID)
	}

	var ctxBuf strings.Builder
	for i, h := range hits {
		fmt.Fprintf(&ctxBuf, "[%d] doc=%s score=%.3f\n%s\n\n",
			i+1, h.DocumentID, h.Score, strings.TrimSpace(h.Content))
	}

	return eval.Output[RAGOutput]{
		Text: resp.Content,
		Extras: RAGOutput{
			RetrievedDocIDs:  docIDs,
			RetrievedContext: ctxBuf.String(),
			Turns:            resp.TotalTurns,
		},
	}, nil
}

// retrievalTap wraps a rag.KnowledgeBase to record every Retrieve
// call's hits.
type retrievalTap struct {
	inner rag.KnowledgeBase
	mu    sync.Mutex
	hits  []rag.Hit
}

func (t *retrievalTap) ID() string { return t.inner.ID() }

func (t *retrievalTap) Ingest(ctx context.Context, docs []rag.Document) error {
	return t.inner.Ingest(ctx, docs)
}

func (t *retrievalTap) Retrieve(
	ctx context.Context,
	query string,
	k int,
) ([]rag.Hit, error) {
	hits, err := t.inner.Retrieve(ctx, query, k)
	if err != nil {
		return hits, err
	}
	t.mu.Lock()
	t.hits = append(t.hits, hits...)
	t.mu.Unlock()
	return hits, nil
}

func (t *retrievalTap) reset() {
	t.mu.Lock()
	t.hits = nil
	t.mu.Unlock()
}

func (t *retrievalTap) allHits() []rag.Hit {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]rag.Hit, len(t.hits))
	copy(out, t.hits)
	return out
}

// retrievalHitAt returns 1 when at least one of the case's
// ExpectedDocIDs is in the top-k retrieved IDs. Off-topic cases
// score 1 (skip).
func retrievalHitAt(k int) eval.Metric[RAGExpectations, RAGOutput] {
	return &retrievalHitMetric{k: k}
}

type retrievalHitMetric struct{ k int }

func (m *retrievalHitMetric) Name() string {
	return fmt.Sprintf("retrieval_hit@%d", m.k)
}

func (m *retrievalHitMetric) Score(
	_ context.Context,
	c eval.Case[RAGExpectations],
	out eval.Output[RAGOutput],
) (eval.Score, error) {
	if c.Extras.OffTopic {
		return eval.Score{Value: 1, Pass: true, Reason: "off-topic skip"}, nil
	}
	retrieved := out.Extras.RetrievedDocIDs
	if m.k > 0 && len(retrieved) > m.k {
		retrieved = retrieved[:m.k]
	}
	for _, want := range c.Extras.ExpectedDocIDs {
		for _, got := range retrieved {
			if got == want {
				return eval.Score{Value: 1, Pass: true,
					Reason: fmt.Sprintf("hit %s in top-%d", want, m.k),
				}, nil
			}
		}
	}
	return eval.Score{Value: 0, Pass: false,
		Reason: fmt.Sprintf("expected one of %v, got %v",
			c.Extras.ExpectedDocIDs, retrieved),
	}, nil
}

// mrr returns the reciprocal rank of the first expected doc in
// the retrieved list. Off-topic cases score 1 (skip).
func mrr() eval.Metric[RAGExpectations, RAGOutput] { return &mrrMetric{} }

type mrrMetric struct{}

func (m *mrrMetric) Name() string { return "mrr" }

func (m *mrrMetric) Score(
	_ context.Context,
	c eval.Case[RAGExpectations],
	out eval.Output[RAGOutput],
) (eval.Score, error) {
	if c.Extras.OffTopic {
		return eval.Score{Value: 1, Pass: true, Reason: "off-topic skip"}, nil
	}
	for i, got := range out.Extras.RetrievedDocIDs {
		for _, want := range c.Extras.ExpectedDocIDs {
			if got == want {
				return eval.Score{
					Value:  1.0 / float64(i+1),
					Pass:   true,
					Reason: fmt.Sprintf("first hit at rank %d", i+1),
				}, nil
			}
		}
	}
	return eval.Score{Value: 0, Pass: false,
		Reason: fmt.Sprintf("no expected doc found; retrieved %v",
			out.Extras.RetrievedDocIDs),
	}, nil
}

// ingestIfNeeded skips embedding for documents whose content hash
// already lives in the store.
func ingestIfNeeded(ctx context.Context, kb rag.KnowledgeBase) error {
	docs, err := loadDocs("../rag-pgvector/data")
	if err != nil {
		return err
	}
	existing, err := loadExistingHashes(ctx, dbURL, kbID)
	if err != nil {
		return err
	}
	toIngest := docs[:0]
	for _, d := range docs {
		hash := contentHash(d.Content)
		if _, ok := existing[d.ID+":"+hash]; ok {
			continue
		}
		d.Metadata = map[string]any{"content_hash": hash}
		toIngest = append(toIngest, d)
	}
	if len(toIngest) == 0 {
		return nil
	}
	t0 := time.Now()
	if err := kb.Ingest(ctx, toIngest); err != nil {
		return err
	}
	fmt.Printf("ingested %d documents in %s\n\n",
		len(toIngest), time.Since(t0).Round(time.Millisecond))
	return nil
}

func loadDocs(dir string) ([]rag.Document, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var docs []rag.Document
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		docs = append(docs, rag.Document{
			ID:      strings.TrimSuffix(e.Name(), ".md"),
			Content: string(body),
		})
	}
	return docs, nil
}

func contentHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func loadExistingHashes(
	ctx context.Context,
	connString, kbID string,
) (map[string]struct{}, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
SELECT DISTINCT document_id, COALESCE(metadata->>'content_hash', '') AS hash
FROM rag_chunks
WHERE kb_id = $1
`, kbID)
	if err != nil {
		return map[string]struct{}{}, nil
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var docID, hash string
		if err := rows.Scan(&docID, &hash); err != nil {
			return nil, err
		}
		if hash == "" {
			continue
		}
		out[docID+":"+hash] = struct{}{}
	}
	return out, rows.Err()
}
