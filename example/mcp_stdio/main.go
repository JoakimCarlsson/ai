package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

func main() {
	ctx := context.Background()

	connectionMethod := os.Getenv("MCP_CONNECTION_METHOD")
	if connectionMethod == "" {
		connectionMethod = "http"
	}

	switch connectionMethod {
	case "stdio":
		stdioExample(ctx)
	case "http":
		httpExample(ctx)
	default:
		log.Fatalf("Invalid connection method: %s. Use 'stdio' or 'http'", connectionMethod)
	}
}

func httpExample(ctx context.Context) {

	mcpServers := map[string]tool.MCPServer{
		"local": {
			Type: tool.MCPStdio,
			URL:  "http://localhost:9349",
		},
	}

	mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
	if err != nil {
		log.Fatal(err)
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	client, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(openaiKey),
		llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	messages := []message.Message{
		message.NewUserMessage("Can you help me understand React hooks? Specifically useState and useEffect. Use Context7 to get the latest documentation."),
	}

	response, err := client.SendMessages(ctx, messages, mcpTools)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response.Content)

	tool.CloseMCPPool()
}

func stdioExample(ctx context.Context) {
	apiKey := os.Getenv("CONTEXT7_API_KEY")
	if apiKey == "" {
		log.Fatal("CONTEXT7_API_KEY environment variable is required")
	}

	mcpServers := map[string]tool.MCPServer{
		"context7": {
			Type:    tool.MCPStdio,
			Command: "npx",
			Args: []string{
				"-y",
				"@upstash/context7-mcp",
				"--api-key",
				apiKey,
			},
		},
	}

	mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
	if err != nil {
		log.Fatal(err)
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	client, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(openaiKey),
		llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	messages := []message.Message{
		message.NewUserMessage("Explain TypeScript utility types like Pick, Omit, and Partial. Use Context7 to fetch the latest TypeScript documentation."),
	}

	response, err := client.SendMessages(ctx, messages, mcpTools)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response.Content)

	tool.CloseMCPPool()
}
