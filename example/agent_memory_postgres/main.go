package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/integrations/pgvector"
	"github.com/joakimcarlsson/ai/integrations/postgres"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

func main() {
	ctx := context.Background()

	connString := "postgres://postgres:password@localhost:5432/example?sslmode=disable"

	embedder, err := embeddings.NewEmbedding(model.ProviderOpenAI,
		embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
	)
	if err != nil {
		log.Fatal(err)
	}

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore, err := postgres.SessionStore(ctx, connString)
	if err != nil {
		log.Fatal(err)
	}

	memoryStore, err := pgvector.MemoryStore(ctx, connString, embedder)
	if err != nil {
		log.Fatal(err)
	}

	agent1 := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with memory capabilities.`),
		agent.WithMemory("alice", memoryStore,
			memory.AutoExtract(),
			memory.AutoDedup(),
		),
		agent.WithSession("conv-1", sessionStore),
	)

	response, err := agent1.Chat(ctx, "Hi! My name is Alice and I love Italian food.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	agent2 := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with memory capabilities.`),
		agent.WithMemory("alice", memoryStore,
			memory.AutoExtract(),
			memory.AutoDedup(),
		),
		agent.WithSession("conv-2", sessionStore),
	)

	response, err = agent2.Chat(ctx, "Can you recommend a restaurant for me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
