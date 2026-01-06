package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/integrations/pgvector"
	"github.com/joakimcarlsson/ai/integrations/postgres"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

var idCounter uint64

func snowflakeID() string {
	ts := time.Now().UnixMilli()
	seq := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%d-%d", ts, seq)
}

func main() {
	ctx := context.Background()

	connStr := "postgres://postgres:password@localhost:5432/example?sslmode=disable"

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
		llm.WithMaxTokens(1024),
	)
	if err != nil {
		log.Fatal(err)
	}

	embedder, err := embeddings.NewEmbedding(model.ProviderOpenAI,
		embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
	)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore, err := postgres.SessionStore(ctx, connStr,
		postgres.WithIDGenerator(snowflakeID),
	)
	if err != nil {
		log.Fatal(err)
	}

	memoryStore, err := pgvector.MemoryStore(ctx, connStr, embedder,
		pgvector.WithIDGenerator(snowflakeID),
	)
	if err != nil {
		log.Fatal(err)
	}

	myAgent := agent.New(llmClient,
		agent.WithSystemPrompt("You are a helpful assistant with memory capabilities."),
		agent.WithMemory("user-alice", memoryStore,
			memory.AutoExtract(),
			memory.AutoDedup(),
		),
		agent.WithSession("conv-custom-ids", sessionStore),
	)

	response, err := myAgent.Chat(ctx, "Hi! My name is Alice and I'm a software engineer who loves Go.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Assistant:", response.Content)

	response, err = myAgent.Chat(ctx, "What do you remember about me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Assistant:", response.Content)
}
