package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

type weatherParams struct {
	Location string `json:"location" desc:"The city name"`
	Units    string `json:"units" desc:"Temperature units" enum:"celsius,fahrenheit" required:"false"`
}

type weatherTool struct{}

func (w *weatherTool) Info() tool.ToolInfo {
	return tool.NewToolInfo("get_weather", "Get the current weather for a location", weatherParams{})
}

func (w *weatherTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input weatherParams
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(fmt.Sprintf("The weather in %s is sunny, 22°C", input.Location)), nil
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

	myAgent := agent.New(llmClient,
		agent.WithSystemPrompt("You are a helpful assistant with access to weather tools."),
		agent.WithTools(&weatherTool{}),
		agent.WithSession("conv-1", agent.FileStore("./sessions")),
	)

	response, err := myAgent.Chat(ctx, "What's the weather in Tokyo? My name is Bob.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
