package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/agent"
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
	ctx = context.WithValue(ctx, "user_id", "larry-barry")

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
		llm.WithMaxTokens(1024),
	)
	if err != nil {
		log.Fatal(err)
	}

	memory, err := NewFileMemory("./data/memories")
	if err != nil {
		log.Fatal(err)
	}

	sessionStore, err := agent.NewFileSessionStore("./data/sessions")
	if err != nil {
		log.Fatal(err)
	}

	larry := agent.New(llmClient, agent.WithSystemPrompt(larryPrompt))

	expert := agent.New(llmClient,
		agent.WithSystemPrompt(expertPrompt),
		agent.WithMemory(memory),
		agent.WithAutoExtract(true),
		agent.WithAutoDedup(true),
	)

	larrySession, err := agent.GetOrCreateSession(ctx, "larry-session", sessionStore)
	if err != nil {
		log.Fatal(err)
	}

	expertSession, err := agent.GetOrCreateSession(ctx, "expert-session", sessionStore)
	if err != nil {
		log.Fatal(err)
	}

	larryMsg := "Hi! I want to learn about cars!"

	for round := 0; round < maxRounds; round++ {
		fmt.Printf("Larry Barry: %s\n", larryMsg)

		fmt.Print("Car Expert: ")
		expertResponse := streamAndCollect(ctx, expert, expertSession, larryMsg)
		fmt.Println()
		fmt.Println()

		if round < maxRounds-1 {
			larryMsg = chat(ctx, larry, larrySession, expertResponse)
		}
	}
}

func streamAndCollect(ctx context.Context, a *agent.Agent, session agent.Session, input string) string {
	var sb strings.Builder
	for event := range a.ChatStream(ctx, session, input) {
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

func chat(ctx context.Context, a *agent.Agent, session agent.Session, input string) string {
	resp, err := a.Chat(ctx, session, input)
	if err != nil {
		return "Wow, that's interesting! Tell me more!"
	}
	return resp.Content
}
