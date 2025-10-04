# Go AI Client Library

[![Go Reference](https://pkg.go.dev/badge/github.com/joakimcarlsson/ai.svg)](https://pkg.go.dev/github.com/joakimcarlsson/ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/joakimcarlsson/ai)](https://goreportcard.com/report/github.com/joakimcarlsson/ai)

A comprehensive, multi-provider Go library for interacting with various AI models through unified interfaces. This library supports Large Language Models (LLMs), embedding models, image generation models, and rerankers from multiple providers including Anthropic, OpenAI, Google, AWS, Voyage AI, xAI, and more, with features like streaming responses, tool calling, structured output, and MCP (Model Context Protocol) integration.

## Features

- **Multi-Provider Support**: Unified interface for 9+ AI providers
- **LLM Support**: Chat completions, streaming, tool calling, structured output
- **Embedding Models**: Text, multimodal, and contextualized embeddings
- **Image Generation**: Text-to-image generation with multiple quality and size options
- **Rerankers**: Document reranking for improved search relevance
- **Streaming Responses**: Real-time response streaming via Go channels
- **Tool Calling**: Native function calling with JSON schema validation
- **Structured Output**: Constrained generation with JSON schemas
- **MCP Integration**: Model Context Protocol support for advanced tooling
- **Multimodal Support**: Text and image inputs across compatible providers
- **Cost Tracking**: Built-in token usage and cost calculation
- **Retry Logic**: Exponential backoff with configurable retry policies
- **Type Safety**: Full Go generics support for compile-time safety

## Supported Providers

### LLM Providers

| Provider | Streaming | Tools | Structured Output | Attachments |
|----------|-----------|-------|-------------------|-------------|
| Anthropic (Claude) | ✅ | ✅ | ❌ | ✅ |
| OpenAI (GPT) | ✅ | ✅ | ✅ | ✅ |
| Google Gemini | ✅ | ✅ | ✅ | ✅ |
| AWS Bedrock | ✅ | ✅ | ❌ | ✅ |
| Azure OpenAI | ✅ | ✅ | ✅ | ✅ |
| Google Vertex AI | ✅ | ✅ | ✅ | ✅ |
| Groq | ✅ | ✅ | ✅ | ✅ |
| OpenRouter | ✅ | ✅ | ✅ | ✅ |
| xAI (Grok) | ✅ | ✅ | ✅ | ✅ |

### Embedding & Reranker Providers

| Provider | Text Embeddings | Multimodal Embeddings | Contextualized Embeddings | Rerankers |
|----------|-----------------|----------------------|---------------------------|-----------|
| Voyage AI | ✅ | ✅ | ✅ | ✅ |
| OpenAI | ✅ | ❌ | ❌ | ❌ |

### Image Generation Providers

| Provider | Models | Quality Options | Size Options |
|----------|--------|-----------------|--------------|
| OpenAI | DALL-E 2, DALL-E 3, GPT Image 1 | standard, hd, low, medium, high | 256x256 to 1792x1024 |
| xAI (Grok) | Grok 2 Image | default | default |
| Google Gemini | Gemini 2.5 Flash Image, Imagen 3, Imagen 4, Imagen 4 Ultra, Imagen 4 Fast | default | Aspect ratios: 1:1, 3:4, 4:3, 9:16, 16:9 |

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

### Embeddings

```go
import (
    "github.com/joakimcarlsson/ai/embeddings"
    "github.com/joakimcarlsson/ai/model"
)

// Create embedding client
embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
)
if err != nil {
    log.Fatal(err)
}

// Generate text embeddings
texts := []string{
    "Hello, world!",
    "This is a test document.",
}

response, err := embedder.GenerateEmbeddings(context.Background(), texts)
if err != nil {
    log.Fatal(err)
}

for i, embedding := range response.Embeddings {
    fmt.Printf("Text: %s\n", texts[i])
    fmt.Printf("Dimensions: %d\n", len(embedding))
    fmt.Printf("First 5 values: %v\n", embedding[:5])
}
```

### Multimodal Embeddings

```go
// Create multimodal embedding client
embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.VoyageMulti3]),
)

// Generate multimodal embeddings (text + image)
multimodalInputs := []embeddings.MultimodalInput{
    {
        Content: []embeddings.MultimodalContent{
            {Type: "text", Text: "This is a banana."},
            {Type: "image_url", ImageURL: "https://example.com/banana.jpg"},
        },
    },
}

response, err := embedder.GenerateMultimodalEmbeddings(context.Background(), multimodalInputs)
```

### Document Reranking

```go
import (
    "github.com/joakimcarlsson/ai/rerankers"
    "github.com/joakimcarlsson/ai/model"
)

// Create reranker client
reranker, err := rerankers.NewReranker(model.ProviderVoyage,
    rerankers.WithAPIKey(""),
    rerankers.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
    rerankers.WithReturnDocuments(true),
)
if err != nil {
    log.Fatal(err)
}

query := "What is machine learning?"
documents := []string{
    "Machine learning is a subset of artificial intelligence.",
    "The weather today is sunny.",
    "Deep learning uses neural networks.",
}

response, err := reranker.Rerank(context.Background(), query, documents)
if err != nil {
    log.Fatal(err)
}

for i, result := range response.Results {
    fmt.Printf("Rank %d (Score: %.4f): %s\n",
        i+1, result.RelevanceScore, result.Document)
}
```

### Image Generation

```go
import (
    "github.com/joakimcarlsson/ai/image_generation"
    "github.com/joakimcarlsson/ai/model"
)

// OpenAI DALL-E 3
client, err := image_generation.NewImageGeneration(
    model.ProviderOpenAI,
    image_generation.WithAPIKey("your-api-key"),
    image_generation.WithModel(model.OpenAIImageGenerationModels[model.DALLE3]),
)
if err != nil {
    log.Fatal(err)
}

response, err := client.GenerateImage(
    context.Background(),
    "A serene mountain landscape at sunset with vibrant colors",
    image_generation.WithSize("1024x1024"),
    image_generation.WithQuality("hd"),
    image_generation.WithResponseFormat("b64_json"),
)
if err != nil {
    log.Fatal(err)
}

imageData, _ := image_generation.DecodeBase64Image(response.Images[0].ImageBase64)
os.WriteFile("image.png", imageData, 0644)

// Google Gemini Imagen 4
client, err := image_generation.NewImageGeneration(
    model.ProviderGemini,
    image_generation.WithAPIKey("your-api-key"),
    image_generation.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
)

response, err := client.GenerateImage(
    context.Background(),
    "A futuristic cityscape at night",
    image_generation.WithSize("16:9"),
    image_generation.WithN(4),
)

// xAI Grok 2 Image
client, err := image_generation.NewImageGeneration(
    model.ProviderXAI,
    image_generation.WithAPIKey("your-api-key"),
    image_generation.WithModel(model.XAIImageGenerationModels[model.XAIGrok2Image]),
)

response, err := client.GenerateImage(
    context.Background(),
    "A robot playing chess",
    image_generation.WithResponseFormat("b64_json"),
)
```

## Project Structure

```
├── example/
│   ├── structured_output/        # Structured output example
│   ├── vision/                   # Multimodal LLM example
│   ├── embeddings/               # Text embedding example
│   ├── multimodal_embeddings/    # Multimodal embedding example
│   ├── contextualized_embeddings/# Contextualized embedding example
│   ├── reranker/                 # Document reranking example
│   ├── image_generation/         # xAI image generation example
│   ├── image_generation_openai/  # OpenAI image generation example
│   └── image_generation_gemini/  # Gemini image generation example
├── image_generation/             # Image generation package
│   ├── image_generation.go      # Main image generation interface
│   ├── openai.go                # OpenAI/xAI implementation
│   └── gemini.go                # Gemini implementation
├── message/                      # Message types and handling
│   ├── base.go                  # Core message structures
│   ├── multimodal.go            # Image/attachment support
│   ├── marshal.go               # Serialization helpers
│   └── text.go                  # Text message utilities
├── model/                        # Model definitions and configurations
│   ├── llm_models.go            # LLM model types
│   ├── embedding_models.go      # Embedding model types
│   ├── image_generation_models.go # Image generation model types
│   ├── voyage.go                # Voyage AI models (embeddings & rerankers)
│   ├── anthropic.go             # Claude models
│   ├── openai.go                # GPT and DALL-E models
│   ├── gemini.go                # Gemini and Imagen models
│   ├── xai.go                   # xAI Grok models
│   └── ...                      # Other provider models
├── providers/                    # AI provider implementations
│   ├── llm.go                   # Main LLM interface
│   ├── embeddings.go            # Embedding interface
│   ├── rerankers.go             # Reranker interface
│   ├── voyage.go                # Voyage embedding implementation
│   ├── voyage_reranker.go       # Voyage reranker implementation
│   ├── anthropic.go             # Anthropic LLM implementation
│   ├── openai.go                # OpenAI LLM implementation
│   ├── retry.go                 # Retry logic
│   └── ...                      # Other providers
├── schema/                       # Structured output schemas
├── tool/                         # Tool calling infrastructure
│   ├── tool.go                  # Base tool interface
│   └── mcp-tools.go             # MCP integration
└── types/                        # Event types for streaming
```

## Configuration Options

### LLM Client Options

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

### Embedding Client Options

```go
embedder, err := embeddings.NewEmbedding(
    model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
    embeddings.WithBatchSize(100),
    embeddings.WithTimeout(30*time.Second),
    embeddings.WithVoyageOptions(
        embeddings.WithInputType("document"),
        embeddings.WithOutputDimension(1024),
        embeddings.WithOutputDtype("float"),
    ),
)
```

### Reranker Client Options

```go
reranker, err := rerankers.NewReranker(
    model.ProviderVoyage,
    rerankers.WithAPIKey(""),
    rerankers.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
    rerankers.WithTopK(10),
    rerankers.WithReturnDocuments(true),
    rerankers.WithTruncation(true),
    rerankers.WithTimeout(30*time.Second),
)
```

### Image Generation Client Options

```go
// OpenAI/xAI
client, err := image_generation.NewImageGeneration(
    model.ProviderOpenAI,
    image_generation.WithAPIKey("your-key"),
    image_generation.WithModel(model.OpenAIImageGenerationModels[model.DALLE3]),
    image_generation.WithTimeout(60*time.Second),
    image_generation.WithOpenAIOptions(
        image_generation.WithOpenAIBaseURL("custom-endpoint"),
    ),
)

// Gemini
client, err := image_generation.NewImageGeneration(
    model.ProviderGemini,
    image_generation.WithAPIKey("your-key"),
    image_generation.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
    image_generation.WithTimeout(60*time.Second),
    image_generation.WithGeminiOptions(
        image_generation.WithGeminiBackend(genai.BackendVertexAI),
    ),
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

#### LLM Models

All LLM models include cost information per million tokens:

```go
model := model.OpenAIModels[model.GPT4o]
fmt.Printf("Input cost: $%.2f per 1M tokens\n", model.CostPer1MIn)
fmt.Printf("Output cost: $%.2f per 1M tokens\n", model.CostPer1MOut)

// Calculate costs from response
response, err := client.SendMessages(ctx, messages, nil)
inputCost := float64(response.Usage.InputTokens) * model.CostPer1MIn / 1_000_000
outputCost := float64(response.Usage.OutputTokens) * model.CostPer1MOut / 1_000_000
```

#### Image Generation Models

Image generation models include pricing by size and quality:

```go
model := model.OpenAIImageGenerationModels[model.DALLE3]

// Pricing structure: size -> quality -> cost
standardCost := model.Pricing["1024x1024"]["standard"]  // $0.04
hdCost := model.Pricing["1024x1024"]["hd"]              // $0.08

fmt.Printf("Standard quality: $%.2f per image\n", standardCost)
fmt.Printf("HD quality: $%.2f per image\n", hdCost)

// GPT Image 1 with multiple quality tiers
gptImageModel := model.OpenAIImageGenerationModels[model.GPTImage1]
lowCost := gptImageModel.Pricing["1024x1024"]["low"]       // $0.011
mediumCost := gptImageModel.Pricing["1024x1024"]["medium"] // $0.042
highCost := gptImageModel.Pricing["1024x1024"]["high"]     // $0.167
```
