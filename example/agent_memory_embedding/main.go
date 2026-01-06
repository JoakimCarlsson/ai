package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

func main() {
	ctx := context.Background()

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

	memStore := memory.FileStore("./memories", embedder)

	agent1 := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with semantic memory.`),
		agent.WithMemory("alice", memStore, memory.AutoDedup()),
		agent.WithSession("conv-1", session.FileStore("./sessions")),
	)

	response, err := agent1.Chat(ctx, "I love hiking in the mountains and prefer vegetarian food.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	agent2 := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with semantic memory.`),
		agent.WithMemory("alice", memStore, memory.AutoDedup()),
		agent.WithSession("conv-2", session.FileStore("./sessions")),
	)

	response, err = agent2.Chat(ctx, "What outdoor activities would you recommend for me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
