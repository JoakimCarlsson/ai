package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

type flightSearchParams struct {
	Origin      string `json:"origin"      desc:"Departure airport code (e.g. SFO)"`
	Destination string `json:"destination" desc:"Arrival airport code (e.g. NRT)"`
}

type flightSearchTool struct{}

func (f *flightSearchTool) Info() tool.ToolInfo {
	return tool.NewToolInfo(
		"flight_search",
		"Search for available flights between two airports",
		flightSearchParams{},
	)
}

func (f *flightSearchTool) Run(
	_ context.Context,
	_ tool.ToolCall,
) (tool.ToolResponse, error) {
	panic("should never be called — autoExecute is disabled")
}

func executeFlightSearch(input string) string {
	var params flightSearchParams
	_ = json.Unmarshal([]byte(input), &params)
	return fmt.Sprintf(
		"Found 3 flights from %s to %s: [UA837 $820 dep 10:30] [JL1 $1150 dep 17:45] [NH7 $980 dep 11:00]",
		params.Origin,
		params.Destination,
	)
}

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

	a := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a travel assistant. Use flight_search to find flights, then recommend the best option.",
		),
		agent.WithTools(&flightSearchTool{}),
		agent.WithAutoExecute(false),
		agent.WithSession("travel-1", session.MemoryStore()),
	)

	resp, err := a.Chat(ctx, "Find me a flight from SFO to Tokyo NRT")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Pending tool calls: %d\n", len(resp.ToolCalls))

	if len(resp.ToolCalls) == 0 {
		fmt.Println(resp.Content)
		return
	}

	var results []message.ToolResult
	for _, tc := range resp.ToolCalls {
		fmt.Printf("Executing: %s(%s)\n", tc.Name, tc.Input)
		output := executeFlightSearch(tc.Input)
		results = append(results, message.ToolResult{
			ToolCallID: tc.ID,
			Name:       tc.Name,
			Content:    output,
		})
	}

	resp, err = a.Continue(ctx, results)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Content)
}
