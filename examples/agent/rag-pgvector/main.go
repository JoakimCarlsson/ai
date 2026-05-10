// Command rag-pgvector demonstrates the rag pipeline backed by
// Postgres + pgvector. Embeddings persist across process restarts.
//
// Setup:
//
//	docker compose up -d
//	export OPENAI_API_KEY=sk-...
//	go run .                            # default question
//	go run . "What is the return policy?"
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
	"time"

	_ "github.com/lib/pq"

	"github.com/joakimcarlsson/ai/agent"
	embedopenai "github.com/joakimcarlsson/ai/embeddings/openai"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rag"
	"github.com/joakimcarlsson/ai/rag/chunkers/fixed"
	pgstore "github.com/joakimcarlsson/ai/rag/store/pgvector"
)

const (
	dbURL = "postgres://rag:rag@localhost:5433/rag?sslmode=disable"
	kbID  = "support-docs"
	dims  = 1536 // text-embedding-3-small
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}
	ctx := context.Background()

	// One-shot schema setup. In production this is a separate deploy
	// step run as a privileged Postgres role; the application path
	// (everything below) only needs DML privileges. Inlined here so
	// the example is one `go run .` to demonstrate.
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

	docs, err := loadDocs("data")
	if err != nil {
		log.Fatalf("load docs: %v", err)
	}

	// Skip ingest when each doc's content hash is already persisted.
	// Persistence is the whole point of using pgvector here; we
	// don't want to re-pay for embeddings on every restart.
	existing, err := loadExistingHashes(ctx, dbURL, kbID)
	if err != nil {
		log.Fatalf("inspect existing chunks: %v", err)
	}

	toIngest := docs[:0]
	for _, d := range docs {
		hash := contentHash(d.Content)
		if _, ok := existing[d.ID+":"+hash]; ok {
			fmt.Printf("skip ingest [%s] (hash %s already persisted)\n",
				d.ID, hash[:8])
			continue
		}
		d.Metadata = map[string]any{"content_hash": hash}
		toIngest = append(toIngest, d)
	}

	if len(toIngest) > 0 {
		t0 := time.Now()
		if err := kb.Ingest(ctx, toIngest); err != nil {
			log.Fatalf("ingest: %v", err)
		}
		fmt.Printf("ingested %d documents in %s\n",
			len(toIngest), time.Since(t0).Round(time.Millisecond))
	} else {
		fmt.Println("all documents already ingested; skipping embed call")
	}

	total, err := countChunks(ctx, dbURL, kbID)
	if err != nil {
		log.Fatalf("count chunks: %v", err)
	}
	fmt.Printf("total chunks in store for kb=%s: %d\n\n", kbID, total)

	llmClient := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT54Mini]),
		llmopenai.WithMaxTokens(512),
	)

	a := agent.New(llmClient,
		agent.WithSystemPrompt(
			"You answer customer-support questions using the supplied knowledge base. "+
				"Cite the source document ID in square brackets after each claim. "+
				"If the answer is not in the knowledge base, say so plainly.",
		),
		agent.WithKnowledgeBase(kb),
		agent.WithTools(rag.SearchTool(kb)),
	)

	question := "How long do I have to return an item, and how do I start the return?"
	if len(os.Args) > 1 {
		question = strings.Join(os.Args[1:], " ")
	}

	t0 := time.Now()
	resp, err := a.Chat(ctx, question)
	if err != nil {
		log.Fatalf("chat: %v", err)
	}
	fmt.Printf("Q: %s\n\nA: %s\n\n(turns=%d, duration=%s)\n",
		question, resp.Content, resp.TotalTurns,
		time.Since(t0).Round(time.Millisecond),
	)
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

// loadExistingHashes returns the set of (doc_id : content_hash)
// pairs already persisted under kbID.
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
		// Table may not yet exist on the very first run.
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

func countChunks(ctx context.Context, connString, kbID string) (int, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var n int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM rag_chunks WHERE kb_id = $1", kbID,
	).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
