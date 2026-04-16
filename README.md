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
- **Audio** — Text-to-speech (ElevenLabs, OpenAI, Google Cloud, Azure Speech) and speech-to-text (OpenAI Whisper, ElevenLabs Scribe, Deepgram, AssemblyAI, Google Cloud)
- **Rerankers** — Document reranking with Voyage AI and Cohere
- **Batch Processing** — Async bulk requests via native batch APIs (OpenAI, Anthropic, Gemini) or bounded concurrent execution for any provider
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

| Provider | LLM | Embeddings | Images | TTS | STT | Rerankers |
|----------|-----|------------|--------|-----|-----|-----------|
| OpenAI | ✅ | ✅ | ✅ | ✅ | ✅ | |
| Anthropic | ✅ | | | | | |
| Google Gemini | ✅ | ✅ | ✅ | | | |
| Google Cloud | | | | ✅ | ✅ | |
| AWS Bedrock | ✅ | ✅ | | | | |
| Azure OpenAI | ✅ | | | | | |
| Azure Speech | | | | ✅ | | |
| Vertex AI | ✅ | | | | | |
| Groq | ✅ | | | | | |
| OpenRouter | ✅ | | | | | |
| xAI | ✅ | | ✅ | | | |
| Voyage AI | | ✅ | | | | ✅ |
| Cohere | ✅ | ✅ | | | | ✅ |
| Mistral | ✅ | ✅ | | | | |
| ElevenLabs | | | | ✅ | ✅ | |
| Deepgram | | | | | ✅ | |
| AssemblyAI | | | | | ✅ | |

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

The agent framework supports [sub-agents](https://joakimcarlsson.github.io/ai/agent/sub-agents/), [handoffs](https://joakimcarlsson.github.io/ai/agent/handoffs/), [fan-out](https://joakimcarlsson.github.io/ai/agent/fan-out/), [team coordination](https://joakimcarlsson.github.io/ai/agent/team-coordination/), [continue/resume](https://joakimcarlsson.github.io/ai/agent/continue/), [context strategies](https://joakimcarlsson.github.io/ai/agent/context-strategies/), [persistent memory](https://joakimcarlsson.github.io/ai/agent/memory/), and [instruction templates](https://joakimcarlsson.github.io/ai/agent/instruction-templates/).

## Batch Processing

Process bulk requests with native batch APIs or bounded concurrency.

```go
import (
    "github.com/joakimcarlsson/ai/batch"
    "github.com/joakimcarlsson/ai/model"
)

// Native batch API — OpenAI, Anthropic, or Gemini
proc, _ := batch.New(
    model.ProviderOpenAI,
    batch.WithAPIKey("your-api-key"),
    batch.WithModel(model.OpenAIModels[model.GPT4o]),
)

// Concurrent fallback — works with any provider
proc, _ := batch.New(
    model.ProviderGroq,
    batch.WithLLM(groqClient),
    batch.WithMaxConcurrency(10),
)

requests := []batch.Request{
    {ID: "q1", Type: batch.RequestTypeChat, Messages: msgs1},
    {ID: "q2", Type: batch.RequestTypeChat, Messages: msgs2},
}

resp, _ := proc.Process(ctx, requests)
for _, r := range resp.Results {
    fmt.Printf("[%s] %s\n", r.ID, r.ChatResponse.Content)
}
```

Per-item error handling, progress callbacks, and async channel-based tracking are all supported. See the [batch processing docs](https://joakimcarlsson.github.io/ai/advanced/batch-processing/) for details.

## Versioning

Each module is versioned independently using path-prefixed git tags. The tag
prefix **must** match the subdirectory path exactly — this is how the Go module
system resolves versions.

| Module | Tag format | Example |
|--------|-----------|---------|
| Root | `vX.Y.Z` | `v0.15.0` |
| postgres | `integrations/postgres/vX.Y.Z` | `integrations/postgres/v0.1.0` |
| sqlite | `integrations/sqlite/vX.Y.Z` | `integrations/sqlite/v1.1.0` |
| pgvector | `integrations/pgvector/vX.Y.Z` | `integrations/pgvector/v0.1.0` |

All modules follow [semantic versioning](https://semver.org). The root module
and integration modules are versioned independently.

## Release Process

Releases follow the AWS SDK v2 pattern: CI on main is the safety net, git tags
drive `go get` resolution, and dated GitHub Releases provide changelogs.

### 1. Ensure main is green

CI must pass on the latest commit before tagging.

### 2. Tag modules that changed

```bash
# Tag a single module (dry-run — creates local tag only)
scripts/release.sh tag -m postgres -v v0.1.0

# Tag and push
make release-tag MODULE=postgres VERSION=v0.1.0
```

For integration modules, the `require` version for the root module in
`integrations/<name>/go.mod` must match the latest published root tag.
The script warns if this is stale.

### 3. Warm the Go module proxy

```bash
scripts/release.sh warm -t integrations/postgres/v0.1.0
```

This ensures the tagged version is immediately available via `go get`.

### 4. Create a dated GitHub Release

```bash
# Dry-run (shows what would be published)
scripts/release.sh release

# Publish
make release-publish
```

This creates a `release-YYYY-MM-DD` tag and a GitHub Release listing all
module versions tagged since the previous release.

## License

See [LICENSE](LICENSE) file.
