# Go AI Client Library

[![Go Reference](https://pkg.go.dev/badge/github.com/joakimcarlsson/ai.svg)](https://pkg.go.dev/github.com/joakimcarlsson/ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/joakimcarlsson/ai)](https://goreportcard.com/report/github.com/joakimcarlsson/ai)

A comprehensive, multi-provider Go library for interacting with various AI models through unified interfaces. Supports LLMs, embeddings, image generation, audio generation, speech-to-text, and rerankers from 10+ providers with streaming, tool calling, structured output, and MCP integration.

**[Documentation](https://joakimcarlsson.github.io/ai)** | **[API Reference](https://pkg.go.dev/github.com/joakimcarlsson/ai)**

## Features

- **Multi-Provider Support** — Unified interface for 10+ AI providers
- **LLM Support** — Chat completions, streaming, tool calling, structured output
- **Agent Framework** — Multi-agent orchestration with sub-agents, handoffs, fan-out, session management, persistent memory, and context strategies
- **Embedding Models** — Text, multimodal, and contextualized embeddings
- **Image Generation** — Text-to-image with OpenAI, Gemini, and xAI
- **Audio** — Text-to-speech (ElevenLabs) and speech-to-text (OpenAI Whisper)
- **Rerankers** — Document reranking for improved search relevance
- **MCP Integration** — Model Context Protocol support for advanced tooling
- **Cost Tracking** — Built-in token and character usage with cost calculation

## Installation

```bash
go get github.com/joakimcarlsson/ai
```

## Quick Start

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
    client, err := llm.NewLLM(
        model.ProviderOpenAI,
        llm.WithAPIKey("api-key"),
        llm.WithModel(model.OpenAIModels[model.GPT4o]),
    )
    if err != nil {
        log.Fatal(err)
    }

    messages := []message.Message{
        message.NewUserMessage("Hello, how are you?"),
    }

    response, err := client.SendMessages(context.Background(), messages, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

## Supported Providers

| Provider | LLM | Embeddings | Images | Audio | STT | Rerankers |
|----------|-----|------------|--------|-------|-----|-----------|
| OpenAI | ✅ | ✅ | ✅ | | ✅ | |
| Anthropic | ✅ | | | | | |
| Google Gemini | ✅ | | ✅ | | | |
| AWS Bedrock | ✅ | | | | | |
| Azure OpenAI | ✅ | | | | | |
| Vertex AI | ✅ | | | | | |
| Groq | ✅ | | | | | |
| OpenRouter | ✅ | | | | | |
| xAI | ✅ | | ✅ | | | |
| Voyage AI | | ✅ | | | | ✅ |
| ElevenLabs | | | | ✅ | | |

## Agent Framework

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/agent/session"
)

myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithTools(&weatherTool{}),
    agent.WithSession("user-123", session.FileStore("./sessions")),
)

response, _ := myAgent.Chat(ctx, "What's the weather in Tokyo?")
```

The agent framework supports [sub-agents](https://joakimcarlsson.github.io/ai/agent/sub-agents/), [handoffs](https://joakimcarlsson.github.io/ai/agent/handoffs/), [fan-out](https://joakimcarlsson.github.io/ai/agent/fan-out/), [continue/resume](https://joakimcarlsson.github.io/ai/agent/continue/), [context strategies](https://joakimcarlsson.github.io/ai/agent/context-strategies/), [persistent memory](https://joakimcarlsson.github.io/ai/agent/memory/), and [instruction templates](https://joakimcarlsson.github.io/ai/agent/instruction-templates/).

## License

See [LICENSE](LICENSE) file.
