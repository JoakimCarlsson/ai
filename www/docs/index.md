# Go AI Client Library

[![Go Reference](https://pkg.go.dev/badge/github.com/joakimcarlsson/ai.svg)](https://pkg.go.dev/github.com/joakimcarlsson/ai)
[![Go Report Card](https://goreportcard.com/badge/github.com/joakimcarlsson/ai)](https://goreportcard.com/report/github.com/joakimcarlsson/ai)

A multi-provider Go library for interacting with AI models through unified
interfaces. Each capability — LLMs, embeddings, TTS, STT, image generation,
rerankers, fill-in-the-middle — is published as its own modality module, and
each vendor implementation is published as its own sub-module. You only pull
the SDK you actually use.

Supported providers: Anthropic, OpenAI, Google (Gemini & Vertex AI), AWS
Bedrock, Azure OpenAI, Voyage AI, Cohere, Mistral, DeepSeek, Groq, OpenRouter,
xAI, ElevenLabs, Deepgram, AssemblyAI, Google Cloud Speech, plus any
OpenAI-compatible endpoint via [BYOM](advanced/byom.md).

## Module structure

The library is published as ~50 independent Go modules. The core split:

- **Modality interfaces** define the contract for each capability:
  `llm`, `embeddings`, `tts`, `stt`, `image`, `rerankers`, `fim`. These pull
  no vendor SDKs.
- **Vendor sub-modules** under each modality carry the SDK and the
  implementation: `llm/anthropic`, `llm/openai`, `embeddings/voyage`,
  `tts/elevenlabs`, etc. You import only the vendors you use.
- **Tier 0 leaves** (`model`, `message`, `tool`, `schema`, `tracing`,
  `prompt`, `types`) are dependency-free building blocks shared across the
  rest.
- **Agent runtime** (`agent`, `agent/memory`) and persistence integrations
  (`agent/memory/{pgvector,postgres,sqlite}`) layer on top.

## Install

You install only the modules you use. For an OpenAI chat client:

```bash
go get github.com/joakimcarlsson/ai/llm
go get github.com/joakimcarlsson/ai/llm/openai
go get github.com/joakimcarlsson/ai/message
go get github.com/joakimcarlsson/ai/model
```

## Quick example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
)

func main() {
    ctx := context.Background()

    client := llmopenai.NewLLM(
        llmopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
        llmopenai.WithMaxTokens(1000),
    )

    response, err := client.SendMessages(ctx, []message.Message{
        message.NewUserMessage("Hello, how are you?"),
    }, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

## Next steps

- [Installation & Quick Start](getting-started/installation.md) — get up and running
- [Provider Overview](providers/overview.md) — every supported provider with capability matrix
- [LLM module](providers/llm.md) — chat, streaming, tools, structured output
- [Agent Framework](agent/overview.md) — multi-agent runtime
- [Bring Your Own Model (BYOM)](advanced/byom.md) — Ollama, LocalAI, custom OpenAI-compatible endpoints
