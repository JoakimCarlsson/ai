// Command rag demonstrates retrieval-augmented grounding for an
// agent. It loads every *.md file under ./data into a knowledge base,
// then asks the question passed on the command line (or a default
// one) and prints the grounded answer.
//
// Usage:
//
//	export OPENAI_API_KEY=sk-...
//	go run . "How do returns work?"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joakimcarlsson/ai/agent"
	embedopenai "github.com/joakimcarlsson/ai/embeddings/openai"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rag"
	"github.com/joakimcarlsson/ai/rag/chunkers/fixed"
	ragmem "github.com/joakimcarlsson/ai/rag/store/memory"
)

const defaultQuestion = "What is your return policy?"

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	ctx := context.Background()

	embedder := embedopenai.NewEmbedding(
		embedopenai.WithAPIKey(apiKey),
		embedopenai.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
	)

	kb := rag.New(
		"docs",
		embedder,
		ragmem.New(),
		rag.WithChunker(fixed.Default),
	)

	docs, err := loadDocs("data")
	if err != nil {
		log.Fatalf("load docs: %v", err)
	}
	if len(docs) == 0 {
		log.Fatal("no markdown files found in ./data")
	}
	if err := kb.Ingest(ctx, docs); err != nil {
		log.Fatalf("ingest: %v", err)
	}

	llmClient := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT54Nano]),
		llmopenai.WithMaxTokens(512),
	)

	a := agent.New(llmClient,
		agent.WithSystemPrompt(
			"You answer questions using the supplied knowledge base. "+
				"Cite the source document ID in square brackets when you use it. "+
				"If the answer is not in the knowledge base, say so plainly.",
		),
		agent.WithKnowledgeBase(kb),
		agent.WithTools(rag.SearchTool(kb)),
	)

	question := defaultQuestion
	if len(os.Args) > 1 {
		question = strings.Join(os.Args[1:], " ")
	}

	resp, err := a.Chat(ctx, question)
	if err != nil {
		log.Fatalf("chat: %v", err)
	}

	fmt.Printf("Q: %s\n\nA: %s\n", question, resp.Content)
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
		path := filepath.Join(dir, e.Name())
		body, err := os.ReadFile(path)
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
