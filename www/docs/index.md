# Go AI Client Library

[![Go Reference](https://pkg.go.dev/badge/github.com/joakimcarlsson/ai.svg)](https://pkg.go.dev/github.com/joakimcarlsson/ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/joakimcarlsson/ai)](https://goreportcard.com/report/github.com/joakimcarlsson/ai)

A comprehensive, multi-provider Go library for interacting with various AI models through unified interfaces. This library supports Large Language Models (LLMs), embedding models, image generation models, audio generation (text-to-speech), and rerankers from multiple providers including Anthropic, OpenAI, Google, AWS, Voyage AI, xAI, ElevenLabs, and more.

## Features

- **Multi-Provider Support** — Unified interface for 10+ AI providers
- **LLM Support** — Chat completions, streaming, tool calling, structured output
- **Agent Framework** — Multi-agent orchestration with sub-agents, handoffs, fan-out, session management, persistent memory, and context strategies
- **Embedding Models** — Text, multimodal, and contextualized embeddings
- **Image Generation** — Text-to-image generation with multiple quality and size options
- **Audio Generation** — Text-to-speech with voice selection and streaming support
- **Speech-to-Text** — Audio transcription and translation with timestamp support
- **Rerankers** — Document reranking for improved search relevance
- **Streaming Responses** — Real-time response streaming via Go channels
- **Tool Calling** — Native function calling with struct-tag schema generation
- **Structured Output** — Constrained generation with JSON schemas
- **MCP Integration** — Model Context Protocol support for advanced tooling
- **Multimodal Support** — Text and image inputs across compatible providers
- **Cost Tracking** — Built-in token and character usage with cost calculation
- **Retry Logic** — Exponential backoff with configurable retry policies
- **Type Safety** — Full Go generics support for compile-time safety

## Quick Install

```bash
go get github.com/joakimcarlsson/ai
```

## Quick Example

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

    client, err := llm.NewLLM(
        model.ProviderOpenAI,
        llm.WithAPIKey("your-api-key"),
        llm.WithModel(model.OpenAIModels[model.GPT4o]),
        llm.WithMaxTokens(1000),
    )
    if err != nil {
        log.Fatal(err)
    }

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

## Next Steps

- [Installation & Quick Start](getting-started/installation.md) — Get up and running
- [Provider Overview](providers/overview.md) — See all supported providers
- [Agent Framework](agent/overview.md) — Build multi-agent systems
- [Advanced Features](advanced/byom.md) — BYOM, MCP, cost tracking

