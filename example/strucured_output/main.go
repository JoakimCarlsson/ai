package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/types"
)

// CodeAnalysis represents the structured output for code analysis.
// Using struct tags to define the JSON schema is the recommended approach.
type CodeAnalysis struct {
	Language   string   `json:"language" desc:"Programming language"`
	Functions  []string `json:"functions" desc:"List of function names"`
	Complexity string   `json:"complexity" desc:"Code complexity level" enum:"low,medium,high"`
}

func main() {
	ctx := context.Background()

	client, err := llm.NewLLM(
		model.ProviderXAI,
		llm.WithAPIKey(""),
		llm.WithModel(model.XAIModels[model.XAIGrok3]),
		llm.WithMaxTokens(1000),
	)
	if err != nil {
		log.Fatal(err)
	}

	if !client.SupportsStructuredOutput() {
		log.Fatal("No structured output support")
	}

	// Recommended: Use struct-based schema generation
	outputSchema := schema.NewStructuredOutputFromStruct(
		"code_analysis",
		"Analyze Go code and extract key information",
		CodeAnalysis{},
	)

	// Alternative (manual approach, for advanced use cases):
	// outputSchema := schema.NewStructuredOutputInfo(
	//     "code_analysis",
	//     "Analyze Go code and extract key information",
	//     map[string]any{
	//         "language": map[string]any{
	//             "type":        "string",
	//             "description": "Programming language",
	//         },
	//         "functions": map[string]any{
	//             "type": "array",
	//             "items": map[string]any{
	//                 "type": "string",
	//             },
	//             "description": "List of function names",
	//         },
	//         "complexity": map[string]any{
	//             "type":        "string",
	//             "description": "Code complexity level",
	//             "enum":        []string{"low", "medium", "high"},
	//         },
	//     },
	//     []string{"language", "functions", "complexity"},
	// )

	messages := []message.Message{
		message.NewUserMessage(`func calculateSum(a, b int) int {
			return a + b
		}
		
		func processData(data []string) error {
			for _, item := range data {
				if len(item) > 100 {
					return fmt.Errorf("item too long")
				}
			}
			return nil
		}`),
	}

	regularExample(ctx, client, messages, outputSchema)
	streamExample(ctx, client, messages, outputSchema)
}

func regularExample(
	ctx context.Context,
	client llm.LLM,
	messages []message.Message,
	outputSchema *schema.StructuredOutputInfo,
) {
	response, err := client.SendMessagesWithStructuredOutput(
		ctx,
		messages,
		nil,
		outputSchema,
	)
	if err != nil {
		log.Fatal(err)
	}

	if response.StructuredOutput != nil {
		fmt.Println("Regular response:", *response.StructuredOutput)
	}
}

func streamExample(
	ctx context.Context,
	client llm.LLM,
	messages []message.Message,
	outputSchema *schema.StructuredOutputInfo,
) {
	stream := client.StreamResponseWithStructuredOutput(
		ctx,
		messages,
		nil,
		outputSchema,
	)

	fmt.Println("\nStreaming response:")
	for event := range stream {
		switch event.Type {
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventComplete:
			if event.Response.StructuredOutput != nil {
				fmt.Println("\nFinal structured output:", *event.Response.StructuredOutput)
			}
		case types.EventError:
			log.Fatal(event.Error)
		}
	}
}
