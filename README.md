# Go AI Client Library

[![Go Reference](https://pkg.go.dev/badge/github.com/joakimcarlsson/ai.svg)](https://pkg.go.dev/github.com/joakimcarlsson/ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/joakimcarlsson/ai)](https://goreportcard.com/report/github.com/joakimcarlsson/ai)

A comprehensive, multi-provider Go library for interacting with various AI models through unified interfaces. This library supports Large Language Models (LLMs), embedding models, image generation models, audio generation (text-to-speech), and rerankers from multiple providers including Anthropic, OpenAI, Google, AWS, Voyage AI, xAI, ElevenLabs, and more, with features like streaming responses, tool calling, structured output, and MCP (Model Context Protocol) integration.

## Features

- **Multi-Provider Support**: Unified interface for 10+ AI providers
- **LLM Support**: Chat completions, streaming, tool calling, structured output
- **Agent Framework**: Stateless agents with session management and persistent memory
- **Embedding Models**: Text, multimodal, and contextualized embeddings
- **Image Generation**: Text-to-image generation with multiple quality and size options
- **Audio Generation**: Text-to-speech with voice selection and streaming support
- **Rerankers**: Document reranking for improved search relevance
- **Streaming Responses**: Real-time response streaming via Go channels
- **Tool Calling**: Native function calling with struct-tag schema generation
- **Structured Output**: Constrained generation with JSON schemas
- **MCP Integration**: Model Context Protocol support for advanced tooling
- **Multimodal Support**: Text and image inputs across compatible providers
- **Cost Tracking**: Built-in token and character usage with cost calculation
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

### Audio Generation Providers (Text-to-Speech)

| Provider | Models | Streaming | Voice Selection | Max Characters |
|----------|--------|-----------|-----------------|----------------|
| ElevenLabs | Multilingual v2, Turbo v2.5, Flash v2.5 | ✅ | ✅ | 10,000 - 40,000 |

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

// Define parameters with struct tags
type WeatherParams struct {
    Location string `json:"location" desc:"City name"`
    Units    string `json:"units" desc:"Temperature units" enum:"celsius,fahrenheit" required:"false"`
}

// Define a custom tool
type WeatherTool struct{}

func (w *WeatherTool) Info() tool.ToolInfo {
    return tool.NewToolInfo("get_weather", "Get current weather for a location", WeatherParams{})
}

func (w *WeatherTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
    var input WeatherParams
    json.Unmarshal([]byte(params.Input), &input)
    return tool.NewTextResponse("Sunny, 22°C"), nil
}

weatherTool := &WeatherTool{}
tools := []tool.BaseTool{weatherTool}

response, err := client.SendMessages(ctx, messages, tools)
```

#### Struct Tag Schema Generation

Generate JSON schemas automatically from Go structs:

```go
type SearchParams struct {
    Query   string   `json:"query" desc:"Search query"`
    Limit   int      `json:"limit" desc:"Max results" required:"false"`
    Filters []string `json:"filters" desc:"Filter tags" required:"false"`
}

// Generates proper JSON schema with types, descriptions, and required fields
info := tool.NewToolInfo("search", "Search documents", SearchParams{})
```

Supported tags:
- `json` - parameter name
- `desc` - parameter description
- `required` - "true" or "false" (non-pointer fields default to required)
- `enum` - comma-separated allowed values

#### Rich Tool Responses

```go
// Text response
tool.NewTextResponse("Result text")

// JSON response (auto-marshals any value)
tool.NewJSONResponse(map[string]any{"status": "ok", "count": 42})

// File/binary response
tool.NewFileResponse(pdfBytes, "application/pdf")

// Image response (base64)
tool.NewImageResponse(base64ImageData)

// Error response
tool.NewTextErrorResponse("Something went wrong")
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

### Agent Framework

The agent package provides stateless agents with automatic tool execution, session management, and persistent memory.

#### Basic Agent

```go
import "github.com/joakimcarlsson/ai/agent"

// Create agent with tools
myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithTools(&weatherTool{}),
)

// Create session store (file-based)
store, _ := agent.NewFileSessionStore("./sessions")

// Get or create a session
session, _ := agent.GetOrCreateSession(ctx, "user-123", store)

// Chat (automatically manages history and tool execution)
response, _ := myAgent.Chat(ctx, session, "What's the weather in Tokyo?")
fmt.Println(response.Content)
```

#### Session Management

```go
// Session store interface - implement for any backend
type SessionStore interface {
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, id string) (Session, error)
    Load(ctx context.Context, id string) (Session, error)
    Delete(ctx context.Context, id string) error
}

// Built-in file session store
store, _ := agent.NewFileSessionStore("./sessions")

// Session helpers
session, _ := agent.CreateSession(ctx, "new-session", store)    // Error if exists
session, _ := agent.LoadSession(ctx, "existing-session", store) // Error if not found
session, _ := agent.GetOrCreateSession(ctx, "session-id", store) // Create or load

// Session operations
messages, _ := session.GetMessages(ctx, nil)  // Get all messages
session.AddMessages(ctx, newMessages)          // Append messages
session.PopMessage(ctx)                        // Remove last message
session.Clear(ctx)                             // Clear all messages
```

#### Persistent Memory

Enable cross-conversation memory for personalization:

```go
// Implement the Memory interface for your storage backend
type Memory interface {
    Store(ctx context.Context, userID, fact string, metadata map[string]any) error
    Search(ctx context.Context, userID, query string, limit int) ([]MemoryEntry, error)
    GetAll(ctx context.Context, userID string, limit int) ([]MemoryEntry, error)
    Delete(ctx context.Context, memoryID string) error
    Update(ctx context.Context, memoryID, fact string, metadata map[string]any) error
}

// Create agent with memory
myAgent := agent.New(llmClient,
    agent.WithSystemPrompt(`You are a personal assistant.
Use store_memory when users share personal information.
Use recall_memories before answering personalized questions.
Use replace_memory when information changes.
Use delete_memory when users ask to forget something.`),
    agent.WithMemory(myMemoryStore),
    agent.WithAutoExtract(true),  // Auto-extract facts from conversations
    agent.WithAutoDedup(true),    // LLM-based memory deduplication
)

// Set user ID in context for memory operations
ctx = context.WithValue(ctx, "user_id", "alice")

response, _ := myAgent.Chat(ctx, session, "My name is Alice and I'm allergic to peanuts.")
// Agent automatically stores this fact

// In a new session, agent can recall memories
response, _ := myAgent.Chat(ctx, newSession, "What restaurants would you recommend?")
// Agent recalls allergy information and personalizes response
```

#### Streaming with Agent

```go
for event := range myAgent.ChatStream(ctx, session, "Tell me a story") {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventToolUseStart:
        fmt.Printf("\nUsing tool: %s\n", event.ToolCall.Name)
    case types.EventComplete:
        fmt.Printf("\nDone! Tokens: %d\n", event.Response.Usage.InputTokens)
    }
}
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

### Audio Generation (Text-to-Speech)

```go
import (
    "github.com/joakimcarlsson/ai/audio"
    "github.com/joakimcarlsson/ai/model"
)

// Create audio generation client
client, err := audio.NewAudioGeneration(
    model.ProviderElevenLabs,
    audio.WithAPIKey("your-api-key"),
    audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
)
if err != nil {
    log.Fatal(err)
}

// Generate audio
response, err := client.GenerateAudio(
    context.Background(),
    "Hello! This is a demonstration of text-to-speech.",
    audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
)
if err != nil {
    log.Fatal(err)
}

// Save to file
os.WriteFile("output.mp3", response.AudioData, 0644)
fmt.Printf("Characters used: %d\n", response.Usage.Characters)
```

#### Custom Voice Settings

```go
// Generate with custom voice settings
response, err := client.GenerateAudio(
    context.Background(),
    "This uses custom voice settings for enhanced expressiveness.",
    audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
    audio.WithStability(0.75),              // 0.0-1.0, higher = more consistent
    audio.WithSimilarityBoost(0.85),        // 0.0-1.0, higher = more similar to original
    audio.WithStyle(0.5),                   // 0.0-1.0, higher = more expressive
    audio.WithSpeakerBoost(true),           // Enhanced speaker similarity
)
```

#### Streaming Audio

```go
// Stream audio in chunks for real-time playback
chunkChan, err := client.StreamAudio(
    context.Background(),
    "This is a streaming audio example.",
    audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
    audio.WithOptimizeStreamingLatency(3), // 0-4, higher = lower latency
)
if err != nil {
    log.Fatal(err)
}

file, _ := os.Create("output_stream.mp3")
defer file.Close()

for chunk := range chunkChan {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    if chunk.Done {
        break
    }
    file.Write(chunk.Data)
}
```

#### List Available Voices

```go
voices, err := client.ListVoices(context.Background())
if err != nil {
    log.Fatal(err)
}

for _, voice := range voices {
    fmt.Printf("%s (%s) - %s\n", voice.Name, voice.VoiceID, voice.Category)
}
```

## Project Structure

```
├── agent/                        # Agent framework
│   ├── agent.go                 # Core agent with tool execution loop
│   ├── session.go               # Session interface and helpers
│   ├── file_session.go          # File-based session storage
│   ├── memory.go                # Memory interface for persistence
│   ├── memory_tools.go          # Built-in memory tools
│   ├── extract.go               # Auto-extraction logic
│   ├── dedup.go                 # Memory deduplication
│   └── options.go               # Agent configuration options
├── example/
│   ├── agent/                   # Basic agent example
│   ├── agent_memory/            # Agent with memory example
│   ├── agent_memory_embedding/  # Agent with vector memory
│   ├── agent_memory_postgres/   # PostgreSQL session & memory
│   ├── structured_output/       # Structured output example
│   ├── vision/                  # Multimodal LLM example
│   ├── embeddings/              # Text embedding example
│   ├── multimodal_embeddings/   # Multimodal embedding example
│   ├── contextualized_embeddings/# Contextualized embedding example
│   ├── reranker/                # Document reranking example
│   ├── image_generation/        # xAI image generation example
│   ├── image_generation_openai/ # OpenAI image generation example
│   ├── image_generation_gemini/ # Gemini image generation example
│   └── audio/                   # Audio generation (TTS) example
├── audio/                       # Audio generation package
│   ├── audio.go                 # Main audio generation interface
│   ├── elevenlabs.go            # ElevenLabs TTS implementation
│   └── elevenlabs_options.go   # ElevenLabs-specific options
├── image_generation/            # Image generation package
│   ├── image_generation.go      # Main image generation interface
│   ├── openai.go                # OpenAI/xAI implementation
│   └── gemini.go                # Gemini implementation
├── message/                     # Message types and handling
│   ├── base.go                  # Core message structures
│   ├── multimodal.go            # Image/attachment support
│   ├── marshal.go               # Serialization helpers
│   └── text.go                  # Text message utilities
├── model/                       # Model definitions and configurations
│   ├── llm_models.go            # LLM model types
│   ├── embedding_models.go      # Embedding model types
│   ├── image_generation_models.go # Image generation model types
│   ├── audio_models.go          # Audio generation model types
│   ├── voyage.go                # Voyage AI models
│   ├── anthropic.go             # Claude models
│   ├── openai.go                # GPT and DALL-E models
│   ├── gemini.go                # Gemini and Imagen models
│   ├── xai.go                   # xAI Grok models
│   └── ...                      # Other provider models
├── providers/                   # AI provider implementations
│   ├── llm.go                   # Main LLM interface
│   ├── embeddings.go            # Embedding interface
│   ├── rerankers.go             # Reranker interface
│   ├── anthropic.go             # Anthropic implementation
│   ├── openai.go                # OpenAI implementation
│   ├── voyage.go                # Voyage implementation
│   ├── retry.go                 # Retry logic
│   └── ...                      # Other providers
├── schema/                      # Structured output schemas
├── tool/                        # Tool calling infrastructure
│   ├── tool.go                  # Base tool interface and helpers
│   ├── schema.go                # Struct-tag schema generation
│   └── mcp-tools.go             # MCP integration
└── types/                       # Event types for streaming
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

### Audio Generation Client Options

```go
client, err := audio.NewAudioGeneration(
    model.ProviderElevenLabs,
    audio.WithAPIKey("your-key"),
    audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
    audio.WithTimeout(30*time.Second),
    audio.WithElevenLabsOptions(
        audio.WithElevenLabsBaseURL("custom-endpoint"),
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

### BYOM (Bring Your Own Model)

Use Ollama, LocalAI, or any OpenAI-compatible inference server with 3 simple steps:

```go
// 1. Create model
llamaModel := model.NewCustomModel(
    model.WithModelID("llama3.2"),
    model.WithAPIModel("llama3.2:latest"),
)

// 2. Register provider
ollama := llm.RegisterCustomProvider("ollama", llm.CustomProviderConfig{
    BaseURL:      "http://localhost:11434/v1",
    DefaultModel: llamaModel,
})

// 3. Use it
client, _ := llm.NewLLM(ollama)
response, _ := client.SendMessages(ctx, messages, nil)
```

**Supported servers**: Ollama, LocalAI, vLLM, LM Studio, or any OpenAI-compatible API.

See `example/byom/main.go` for complete example.

### MCP (Model Context Protocol) Integration

This library integrates with the official [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk) to provide seamless access to MCP servers and their tools.

#### Stdio Connection (subprocess)

```go
import "github.com/joakimcarlsson/ai/tool"

// Configure MCP servers
mcpServers := map[string]tool.MCPServer{
    "filesystem": {
        Type:    tool.MCPStdio,
        Command: "npx",
        Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/directory"},
        Env:     []string{"NODE_ENV=production"}, // optional
    },
}

// Get MCP tools (connects to servers and retrieves available tools)
mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
if err != nil {
    log.Fatal(err)
}

// Use MCP tools with LLM
response, err := client.SendMessages(ctx, messages, mcpTools)

// Clean up connections when done
defer tool.CloseMCPPool()
```

#### SSE Connection (HTTP)

```go
mcpServers := map[string]tool.MCPServer{
    "remote": {
        Type: tool.MCPSse,
        URL:  "https://your-mcp-server.com/mcp",
        Headers: map[string]string{
            "Authorization": "Bearer your-token",
        },
    },
}

mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
if err != nil {
    log.Fatal(err)
}

// Use tools...
defer tool.CloseMCPPool()
```

#### Complete Example

```go
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

    // Configure MCP server
    mcpServers := map[string]tool.MCPServer{
        "context7": {
            Type:    tool.MCPStdio,
            Command: "npx",
            Args: []string{
                "-y",
                "@upstash/context7-mcp",
                "--api-key",
                os.Getenv("CONTEXT7_API_KEY"),
            },
        },
    }

    // Get MCP tools
    mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
    if err != nil {
        log.Fatal(err)
    }
    defer tool.CloseMCPPool()

    // Create LLM client
    client, err := llm.NewLLM(
        model.ProviderOpenAI,
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Use MCP tools with LLM
    messages := []message.Message{
        message.NewUserMessage("Explain React hooks using Context7 to fetch the latest documentation"),
    }

    response, err := client.SendMessages(ctx, messages, mcpTools)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

**Features:**
- Supports both stdio (subprocess) and SSE (HTTP) transports
- Connection pooling for efficient reuse of MCP server connections
- Automatic tool discovery and registration
- Compatible with all official MCP servers
- Tools are namespaced with server name (e.g., `context7_search`)
- Graceful cleanup with `CloseMCPPool()`

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

#### Audio Generation Models

Audio generation models use character-based pricing:

```go
model := model.ElevenLabsAudioModels[model.ElevenTurboV2_5]

fmt.Printf("Cost per 1M chars: $%.2f\n", model.CostPer1MChars)
fmt.Printf("Max characters per request: %d\n", model.MaxCharacters)
fmt.Printf("Supports streaming: %v\n", model.SupportsStreaming)

// Calculate costs from response
response, err := client.GenerateAudio(ctx, text, audio.WithVoiceID("voice-id"))
cost := float64(response.Usage.Characters) * model.CostPer1MChars / 1_000_000
fmt.Printf("Cost: $%.4f\n", cost)
```
