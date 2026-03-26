// Example confirmation demonstrates requiring human approval before executing sensitive tools.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

type deleteParams struct {
	Table string `json:"table" desc:"Table to delete from"`
	Where string `json:"where" desc:"WHERE clause for deletion"`
}

type deleteTool struct{}

func (d *deleteTool) Info() tool.Info {
	info := tool.NewInfo(
		"delete_records",
		"Delete records from a database table",
		deleteParams{},
	)
	info.RequireConfirmation = true
	return info
}

func (d *deleteTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input deleteParams
	if err := json.Unmarshal(
		[]byte(params.Input),
		&input,
	); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(
		fmt.Sprintf(
			"Deleted records from %s where %s",
			input.Table,
			input.Where,
		),
	), nil
}

type queryParams struct {
	SQL string `json:"sql" desc:"SQL query to execute"`
}

type queryTool struct{}

func (q *queryTool) Info() tool.Info {
	return tool.NewInfo(
		"run_query",
		"Execute a read-only SQL query",
		queryParams{},
	)
}

func (q *queryTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input queryParams
	if err := json.Unmarshal(
		[]byte(params.Input),
		&input,
	); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(
		fmt.Sprintf(
			"Query results for: %s\n[{id: 1, name: 'Alice'}, {id: 2, name: 'Bob'}]",
			input.SQL,
		),
	), nil
}

func cliConfirmationProvider(
	_ context.Context,
	req tool.ConfirmationRequest,
) (bool, error) {
	fmt.Println()
	fmt.Printf(
		"Tool %q wants to execute with input:\n  %s\n",
		req.ToolName,
		req.Input,
	)
	if req.Hint != "" {
		fmt.Printf("  Hint: %s\n", req.Hint)
	}
	fmt.Print("Approve? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y", nil
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

	myAgent := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a database assistant. Use run_query for reads and delete_records for deletions.",
		),
		agent.WithTools(&queryTool{}, &deleteTool{}),
		agent.WithConfirmationProvider(cliConfirmationProvider),
	)

	response, err := myAgent.Chat(
		ctx,
		"List all users, then delete users where name = 'Bob'",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println(response.Content)
}
