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

type searchParams struct {
	Query string `json:"query" desc:"The search query"`
}

type searchTool struct{}

func (s *searchTool) Info() tool.ToolInfo {
	return tool.NewToolInfo("web_search", "Search the web for information", searchParams{})
}

func (s *searchTool) Run(_ context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input searchParams
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(fmt.Sprintf("Search results for %q: [Result 1: relevant info] [Result 2: more details]", input.Query)), nil
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

	researcher := agent.New(llmClient,
		agent.WithSystemPrompt("You are a research assistant. Use the web_search tool to find information, then summarize your findings concisely."),
		agent.WithTools(&searchTool{}),
	)

	writer := agent.New(llmClient,
		agent.WithSystemPrompt("You are a skilled writer. Take the provided information and write a clear, engaging summary."),
	)

	orchestrator := agent.New(llmClient,
		agent.WithSystemPrompt("You are a coordinator. When asked about a topic, first use the researcher to gather information, then use the writer to produce a polished summary."),
		agent.WithSubAgents(
			agent.SubAgentConfig{
				Name:        "researcher",
				Description: "Research a topic using web search. Send a research question as the task.",
				Agent:       researcher,
			},
			agent.SubAgentConfig{
				Name:        "writer",
				Description: "Write a polished summary from provided information. Send the raw info as the task.",
				Agent:       writer,
			},
		),
	)

	response, err := orchestrator.Chat(ctx, "Tell me about the Go programming language.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
