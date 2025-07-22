# Go LLM Client Library

A comprehensive, multi-provider Go library for interacting with various Large Language Models (LLMs) through a unified interface. This library supports multiple AI providers including Anthropic, OpenAI, Google, AWS, and more, with features like streaming responses, tool calling, structured output, and MCP (Model Context Protocol) integration.

## Features

- **Multi-Provider Support**: Unified interface for 9+ AI providers
- **Streaming Responses**: Real-time response streaming via Go channels
- **Tool Calling**: Native function calling with JSON schema validation
- **Structured Output**: Constrained generation with JSON schemas
- **MCP Integration**: Model Context Protocol support for advanced tooling
- **Multimodal Support**: Text and image inputs across compatible providers
- **Cost Tracking**: Built-in token usage and cost calculation
- **Retry Logic**: Exponential backoff with configurable retry policies
- **Type Safety**: Full Go generics support for compile-time safety

## Supported Providers

| Provider | Streaming | Tools | Structured Output | Attachments |
|----------|-----------|-------|-------------------|-------------|
| Anthropic (Claude) | ✅ | ✅ | ❌ | ✅ |
| OpenAI (GPT) | ✅ | ✅ | ✅ | ✅ |
| Google Gemini | ✅ | ✅ | ✅ | ✅ |
| AWS Bedrock | ✅ | ✅ | Varies | ✅ |
| Azure OpenAI | ✅ | ✅ | ✅ | ✅ |
| Google Vertex AI | ✅ | ✅ | ✅ | ✅ |
| Groq | ✅ | ✅ | ✅ | ✅ |
| OpenRouter | ✅ | ✅ | ✅ | ✅ |
| xAI (Grok) | ✅ | ✅ | ✅ | ✅ |

## Installation

```bash
go get github.com/joakimcarlsson/ai
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
)

func main() {
    ctx := context.Background()

    // Create client
    client, err := llm.NewLLM(
        model.ProviderOpenAI,
        llm.WithAPIKey("your-api-key"),
        llm.WithModel(model.OpenAIModels[model.GPT4o]),
        llm.WithMaxTokens(1000),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Send message
    messages := []message.Message{
        message.NewUserMessage("Hello, how are you?"),
    }

    response, err := client.SendMessages(ctx, messages, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

### Streaming Responses

```go
stream := client.StreamResponse(ctx, messages, nil)

for event := range stream {
    switch event.Type {
    case types.EventTypeContentDelta:
        fmt.Print(event.Content)
    case types.EventTypeFinal:
        fmt.Printf("\nTokens used: %d\n", event.Response.Usage.InputTokens)
    case types.EventTypeError:
        log.Fatal(event.Error)
    }
}
```

### Structured Output

```go
type CodeAnalysis struct {
    Language   string   `json:"language"`
    Functions  []string `json:"functions"`
    Complexity string   `json:"complexity"`
}

// Define schema
schema := &schema.StructuredOutputInfo{
    Name:        "code_analysis",
    Description: "Analyze code structure",
    Parameters: map[string]any{
        "language": map[string]any{
            "type":        "string",
            "description": "Programming language",
        },
        "functions": map[string]any{
            "type": "array",
            "items": map[string]any{"type": "string"},
            "description": "List of function names",
        },
        "complexity": map[string]any{
            "type": "string",
            "enum": []string{"low", "medium", "high"},
        },
    },
    Required: []string{"language", "functions", "complexity"},
}

response, err := client.SendMessagesWithStructuredOutput(ctx, messages, nil, schema)
if err != nil {
    log.Fatal(err)
}

// Parse the structured output
var analysis CodeAnalysis
json.Unmarshal([]byte(*response.StructuredOutput), &analysis)
```

### Tool Calling

```go
import "github.com/joakimcarlsson/ai/tool"

// Define a custom tool
type WeatherTool struct{}

func (w *WeatherTool) Info() tool.ToolInfo {
    return tool.ToolInfo{
        Name:        "get_weather",
        Description: "Get current weather for a location",
        Parameters: map[string]any{
            "location": map[string]any{
                "type":        "string",
                "description": "City name",
            },
        },
        Required: []string{"location"},
    }
}

func (w *WeatherTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
    return tool.ToolResponse{
        Type:    tool.ToolResponseTypeText,
        Content: "Sunny, 22°C",
    }, nil
}

weatherTool := &WeatherTool{}
tools := []tool.BaseTool{weatherTool}

response, err := client.SendMessages(ctx, messages, tools)
```

### Multimodal (Images)

```go
// Load image
imageData, err := os.ReadFile("image.png")
if err != nil {
    log.Fatal(err)
}

// Create message with attachment
msg := message.NewUserMessage("What's in this image?")
msg.AddAttachment(message.Attachment{
    MIMEType: "image/png",
    Data:     imageData,
})

messages := []message.Message{msg}
response, err := client.SendMessages(ctx, messages, nil)
```

## Project Structure

```
├── example/
│   └── structured_output/     # Example implementations
├── message/                   # Message types and handling
│   ├── base.go               # Core message structures
│   ├── multimodal.go         # Image/attachment support
│   ├── marshal.go            # Serialization helpers
│   └── text.go               # Text message utilities
├── model/                     # Model definitions and configurations
│   ├── models.go             # Base model structure
│   ├── anthropic.go          # Claude models
│   ├── openai.go             # GPT models
│   ├── gemini.go             # Gemini models
│   └── ...                   # Other provider models
├── providers/                 # LLM provider implementations
│   ├── llm.go                # Main LLM interface
│   ├── anthropic.go          # Anthropic implementation
│   ├── openai.go             # OpenAI implementation
│   ├── retry.go              # Retry logic
│   └── ...                   # Other providers
├── schema/                    # Structured output schemas
├── tool/                      # Tool calling infrastructure
│   ├── tool.go               # Base tool interface
│   └── mcp-tools.go          # MCP integration
└── types/                     # Event types for streaming
```

## Configuration Options

### Client Options

```go
client, err := llm.NewLLM(
    model.ProviderOpenAI,
    llm.WithAPIKey("your-key"),
    llm.WithModel(model.OpenAIModels[model.GPT4o]),
    llm.WithMaxTokens(2000),
    llm.WithTemperature(0.7),
    llm.WithTopP(0.9),
    llm.WithTimeout(30*time.Second),
    llm.WithStopSequences("STOP", "END"),
)
```

### Provider-Specific Options

```go
// Anthropic options
llm.WithAnthropicOptions(
    llm.WithAnthropicBeta("beta-feature"),
)

// OpenAI options
llm.WithOpenAIOptions(
    llm.WithOpenAIBaseURL("custom-endpoint"),
    llm.WithOpenAIExtraHeaders(map[string]string{
        "Custom-Header": "value",
    }),
)
```

## Advanced Features

### MCP (Model Context Protocol) Integration

```go
import "github.com/joakimcarlsson/ai/tool"

// Configure MCP server
mcpServer := tool.MCPServer{
    Command: "npx",
    Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/directory"},
    Type:    tool.MCPStdio,
}

// Create MCP tools
mcpTools, err := tool.NewMCPTools("filesystem", mcpServer)
if err != nil {
    log.Fatal(err)
}

// Use MCP tools with LLM
response, err := client.SendMessages(ctx, messages, mcpTools)
```

### Cost Tracking

All models include cost information per million tokens:

```go
model := model.OpenAIModels[model.GPT4o]
fmt.Printf("Input cost: $%.2f per 1M tokens\n", model.CostPer1MIn)
fmt.Printf("Output cost: $%.2f per 1M tokens\n", model.CostPer1MOut)

// Calculate costs from response
response, err := client.SendMessages(ctx, messages, nil)
inputCost := float64(response.Usage.InputTokens) * model.CostPer1MIn / 1_000_000
outputCost := float64(response.Usage.OutputTokens) * model.CostPer1MOut / 1_000_000
```
