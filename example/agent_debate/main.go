package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/integrations/pgvector"
	"github.com/joakimcarlsson/ai/integrations/postgres"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
)

const larryPrompt = `You are Larry Barry, an emotional and enthusiastic person who is trying to learn about cars.

Your personality:
- You get VERY excited and use caps for emphasis
- You're a bit anxious about cars but determined to learn
- You live in Minnesota where it gets freezing cold
- You work as a nurse and drive 30 miles to work
- Your budget for a car is around $35,000
- You're jealous of your neighbor's Tesla

You are chatting with a car expert AI to learn about cars. Keep your responses fairly short (2-3 sentences).
Share personal details naturally as they come up in conversation.
Ask follow-up questions based on what the AI tells you.`

const expertPrompt = `You are a friendly and patient car expert helping users learn about cars.

You should:
- Be empathetic and acknowledge the user's emotions
- Give clear, simple explanations about cars
- Keep responses concise (3-5 sentences max)
- Reference things the user told you earlier`

const maxRounds = 15

func main() {
	ctx := context.Background()

	connStr := "postgres://postgres:password@localhost:5432/example?sslmode=disable"

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
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

	sessionStore, err := postgres.SessionStore(ctx, connStr)
	if err != nil {
		log.Fatal(err)
	}

	memoryStore, err := pgvector.MemoryStore(ctx, connStr, embedder)
	if err != nil {
		log.Fatal(err)
	}

	larry := agent.New(llmClient,
		agent.WithSystemPrompt(larryPrompt),
		agent.WithSession("larry-session", sessionStore),
	)

	expert := agent.New(llmClient,
		agent.WithSystemPrompt(expertPrompt),
		agent.WithMemory("larry-barry", memoryStore,
			memory.AutoExtract(),
			memory.AutoDedup(),
		),
		agent.WithSession("expert-session", sessionStore),
	)

	expertResponse := ""

	for round := 0; round < maxRounds; round++ {
		fmt.Print("Larry Barry: ")
		var larryMsg string
		if round == 0 {
			larryMsg = streamAndCollect(ctx, larry, "Start the conversation by introducing yourself and saying you want to learn about cars.")
		} else {
			larryMsg = streamAndCollect(ctx, larry, expertResponse)
		}
		fmt.Println()
		fmt.Println()

		fmt.Print("Car Expert: ")
		expertResponse = streamAndCollect(ctx, expert, larryMsg)
		fmt.Println()
		fmt.Println()
	}
}

func streamAndCollect(ctx context.Context, a *agent.Agent, input string) string {
	var sb strings.Builder
	for event := range a.ChatStream(ctx, input) {
		switch event.Type {
		case types.EventContentDelta:
			fmt.Print(event.Content)
			sb.WriteString(event.Content)
		case types.EventError:
			fmt.Printf("[Error: %v]", event.Error)
		}
	}
	return sb.String()
}
