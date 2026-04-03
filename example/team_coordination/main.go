// Example team_coordination demonstrates multi-agent team collaboration with peer-to-peer messaging.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/types"
)

func main() {
	ctx := context.Background()

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	researcher := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			`You are a research specialist on a team. Your job is to research a given topic.
When done, use send_message to share your findings with the team lead.
Also use the task board to claim and complete any tasks assigned to you.`,
		),
	)

	reviewer := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			`You are a review specialist on a team. Your job is to review findings from other teammates.
Use read_messages to check for incoming findings, then send your review back via send_message.
Also use the task board to claim and complete any tasks assigned to you.`,
		),
	)

	lead := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			`You are a team lead coordinating research. Your workflow:
1. Create tasks on the board for each aspect of the topic
2. Spawn teammates to handle research and review
3. Read messages from teammates to collect their findings
4. Synthesize the results into a final answer`,
		),
		agent.WithTeam(team.Config{
			Name:    "research-team",
			MaxSize: 3,
		}),
		agent.WithCoordinatorMode(),
		agent.WithTeammateTemplates(map[string]*agent.Agent{
			"researcher": researcher,
			"reviewer":   reviewer,
		}),
	)

	fmt.Println("Streaming team coordination:")
	fmt.Println()

	for event := range lead.ChatStream(ctx, "Research the pros and cons of microservices vs monoliths.") {
		switch event.Type {
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventTeammateSpawned:
			fmt.Printf("\n[Teammate spawned: %s]\n", event.AgentName)
		case types.EventTeammateComplete:
			fmt.Printf("\n[Teammate completed: %s]\n", event.AgentName)
		case types.EventTeammateError:
			fmt.Printf(
				"\n[Teammate error: %s — %v]\n",
				event.AgentName,
				event.Error,
			)
		case types.EventError:
			log.Fatal(event.Error)
		}
		if event.ToolResult != nil {
			fmt.Printf(
				"\n[Tool: %s → %s]\n",
				event.ToolResult.ToolName,
				truncate(event.ToolResult.Output, 120),
			)
		}
	}
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
