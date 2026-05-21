# Go AI Client Library

[![CI](https://github.com/joakimcarlsson/ai/actions/workflows/ci.yml/badge.svg)](https://github.com/joakimcarlsson/ai/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go)](https://go.dev/)

> **Migrating?** [MIGRATION.md](MIGRATION.md) covers two transitions: `v0.18.x → v0.1.0` (single module split into ~50 per-vendor modules) and `v0.1.x → v0.2.0` (`memory` and `session` lifted out of `agent/` to top-level modules).

A multi-provider Go library for AI: LLMs, embeddings, image generation, TTS,
STT, rerankers, and fill-in-the-middle. Each capability is a modality module
and each vendor implementation is its own sub-module — you import only the
SDKs you actually use.

**[Documentation](https://joakimcarlsson.github.io/ai)**

## Features

- **Per-vendor modules** — Pull only the SDKs you need; no transitive bloat
- **LLM** — Chat, streaming, tool calling, structured output, reasoning
- **Agent framework** — Sub-agents, handoffs, fan-out, sessions, persistent memory, context strategies
- **Voice agent** — Low-latency streaming STT → LLM → TTS pipeline with barge-in, filler audio, tool-call sounds, sessions, hooks, handoffs, toolsets, and memory
- **Embeddings** — Text, multimodal, and contextualized
- **Image generation** — OpenAI, Gemini, xAI
- **Audio** — TTS (ElevenLabs, OpenAI, Google Cloud, Azure Speech) and STT (OpenAI Whisper, ElevenLabs Scribe, Deepgram, AssemblyAI, Google Cloud)
- **Rerankers** — Voyage AI, Cohere
- **Fill-in-the-middle** — Mistral, DeepSeek
- **Batch processing** — Native batch APIs (OpenAI, Anthropic, Gemini) or bounded concurrency for any provider
- **MCP integration** — Model Context Protocol tooling
- **OpenTelemetry tracing** — GenAI semantic conventions across every provider call
- **Cost tracking** — Token / character usage with cost calculation

## Module structure

The library is published as ~50 independent Go modules organised by tier:

- **Tier 0 leaves** — `model`, `message`, `tool`, `schema`, `tracing`, `prompt`, `types` (no vendor SDKs)
- **Tier 1 modality interfaces** — `llm`, `embeddings`, `tts`, `stt`, `image`, `rerankers`, `fim` (no vendor SDKs)
- **Tier 2 vendor implementations** — `llm/openai`, `llm/anthropic`, `embeddings/voyage`, `tts/elevenlabs`, etc. (carry the vendor SDK)
- **Tier 3 utilities** — `tokens/{sliding,truncate,summarize}`, `batch/{openai,anthropic,gemini,concurrent}`
- **Tier 4 agent runtime** — `agent`, `agent/team`, `session`, `memory`, `voice`
- **Tier 5 persistence** — `memory/{pgvector,postgres,sqlite}`

See the **[full module list](https://joakimcarlsson.github.io/ai/modules/)** for every package, its purpose, and the vendor SDK it carries.

## Installation

You install only the modules you use. For an OpenAI chat client:

```bash
go get github.com/joakimcarlsson/ai/llm
go get github.com/joakimcarlsson/ai/llm/openai
go get github.com/joakimcarlsson/ai/message
go get github.com/joakimcarlsson/ai/model
```

## Quick start

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
    client := llmopenai.NewLLM(
        llmopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
    )

    response, err := client.SendMessages(context.Background(), []message.Message{
        message.NewUserMessage("Hello, how are you?"),
    }, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

## Supported providers

| Provider | LLM | Embeddings | Images | TTS | STT | Rerankers | FIM |
|----------|-----|------------|--------|-----|-----|-----------|-----|
| OpenAI | ✅ | ✅ | ✅ | ✅ | ✅ | | |
| Anthropic | ✅ | | | | | | |
| Google Gemini | ✅ | ✅ | ✅ | | | | |
| Google Cloud | | | | ✅ | ✅ | | |
| AWS Bedrock | ✅ | ✅ | | | | | |
| Azure OpenAI | ✅ | | | | | | |
| Azure Speech | | | | ✅ | | | |
| Vertex AI | ✅ | | | | | | |
| Groq | ✅ | | | | | | |
| OpenRouter | ✅ | | | | | | |
| xAI | ✅ | | ✅ | | | | |
| Voyage AI | | ✅ | | | | ✅ | |
| Cohere | ✅ | ✅ | | | | ✅ | |
| Mistral | ✅ | ✅ | | | | | ✅ |
| DeepSeek | ✅ | | | | | | ✅ |
| ElevenLabs | | | | ✅ | ✅ | | |
| Deepgram | | | | | ✅ | | |
| AssemblyAI | | | | | ✅ | | |

Plus any OpenAI-compatible endpoint via [BYOM](https://joakimcarlsson.github.io/ai/advanced/byom/).

## Agent framework

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/session"
)

myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithTools(&weatherTool{}),
    agent.WithSession("user-123", session.FileStore("./sessions")),
)

response, _ := myAgent.Chat(ctx, "What's the weather in Tokyo?")
```

The agent framework supports [sub-agents](https://joakimcarlsson.github.io/ai/agent/sub-agents/), [handoffs](https://joakimcarlsson.github.io/ai/agent/handoffs/), [fan-out](https://joakimcarlsson.github.io/ai/agent/fan-out/), [team coordination](https://joakimcarlsson.github.io/ai/agent/team-coordination/), [continue/resume](https://joakimcarlsson.github.io/ai/agent/continue/), [context strategies](https://joakimcarlsson.github.io/ai/agent/context-strategies/), [persistent memory](https://joakimcarlsson.github.io/ai/memory/), and [instruction templates](https://joakimcarlsson.github.io/ai/agent/instruction-templates/).

## Voice agent

`voice/` ships a streaming STT → LLM → TTS pipeline for building low-latency, voice-first conversational agents. Pluggable providers — bring any `stt.SpeechToText`, `llm.LLM`, and `tts.Generation` implementation.

```go
import (
    "github.com/joakimcarlsson/ai/session"
    "github.com/joakimcarlsson/ai/voice"
)

agent := voice.New(llmClient, sttClient, ttsClient,
    voice.WithSystemPrompt("You are a concise voice assistant."),
    voice.WithTools(myTool),
    voice.WithBargeIn(voice.BargeInInterrupt),
    voice.WithSession("user-42", session.MemoryStore()),
)

conv, _ := agent.StartConversation(ctx, audioTransport)
for evt := range conv.Events() {
    // observe transcripts, tool calls, deltas, etc.
}
```

The voice agent supports [barge-in](https://joakimcarlsson.github.io/ai/voice/overview/#barge-in), [filler audio](https://joakimcarlsson.github.io/ai/voice/overview/#filler-audio) and [tool-call sounds](https://joakimcarlsson.github.io/ai/voice/overview/#tool-call-sounds) for slow first tokens / tool execution, [sessions](https://joakimcarlsson.github.io/ai/voice/overview/#session-persistence), [context strategies](https://joakimcarlsson.github.io/ai/voice/overview/#context-window-management), [hooks](https://joakimcarlsson.github.io/ai/voice/overview/#hooks), [handoffs](https://joakimcarlsson.github.io/ai/voice/overview/#handoffs), [toolsets](https://joakimcarlsson.github.io/ai/voice/overview/#toolsets), and [memory](https://joakimcarlsson.github.io/ai/voice/overview/#memory). Four runnable end-to-end examples under [`examples/voice/`](examples/voice/): `web` (kitchen-sink), `handoff`, `toolsets`, `memory`.

## Batch processing

Each batch backend is its own module. Native batch APIs submit a single async
job; the concurrent runner wraps an existing client with bounded concurrency.

```go
import (
    "github.com/joakimcarlsson/ai/batch"
    batchopenai "github.com/joakimcarlsson/ai/batch/openai"
)

proc := batchopenai.NewProcessor(
    batchopenai.WithAPIKey("your-api-key"),
    batchopenai.WithModel(model.OpenAIModels[model.GPT4o]),
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

Per-item error handling, progress callbacks, and async channel-based tracking
are all supported. See the [batch processing docs](https://joakimcarlsson.github.io/ai/advanced/batch-processing/).

## Workspace setup

The repo is a pure Go workspace with no root module. To work locally:

```bash
git clone https://github.com/joakimcarlsson/ai
cd ai
cp go.work.example go.work   # go.work is gitignored
go build ./...
```

`go.work.example` is the canonical workspace file checked into git;
contributors copy or symlink it to `go.work`.

## Versioning

Each module is versioned independently using path-prefixed git tags. The tag
prefix **must** match the subdirectory path exactly — this is how the Go module
system resolves versions.

| Module | Tag format | Example |
|--------|-----------|---------|
| llm/openai | `llm/openai/vX.Y.Z` | `llm/openai/v0.1.0` |
| embeddings/voyage | `embeddings/voyage/vX.Y.Z` | `embeddings/voyage/v0.1.0` |
| agent | `agent/vX.Y.Z` | `agent/v0.2.0` |
| memory/pgvector | `memory/pgvector/vX.Y.Z` | `memory/pgvector/v0.1.0` |

All modules follow [semantic versioning](https://semver.org).

## Release process

Releases follow the AWS SDK v2 pattern: CI on main is the safety net, git tags
drive `go get` resolution, and dated GitHub Releases provide changelogs.

### 1. Ensure main is green

CI must pass on the latest commit before tagging.

### 2. Tag modules that changed

```bash
# List every module
scripts/release.sh modules

# Tag a single module (dry-run — creates local tag only)
scripts/release.sh tag -m llm/openai -v v0.1.0

# Tag and push
make release-tag MODULE=llm/openai VERSION=v0.1.0
```

The script verifies the module's `go.mod` exists and the tag prefix matches
the directory path.

### 3. Cascade to direct consumers (when bumping a shared module)

When the changed module is a shared dep that surfaces user-facing symbols
(`model`, `message`, `memory`, etc.), open a branch and bump the `require`
line in every module a typical user `go get`s to access the new symbols.
For example, a `model` change adding new Gemini constants cascades to
`llm/gemini`, `image/gemini`, `embeddings/gemini`, `batch/gemini`, and
`llm/vertexai`, but not to unrelated providers (`llm/openai`,
`llm/anthropic`) or to umbrella modules that take a user-built client
(`agent`, `voice`).

```bash
cd llm/gemini && go mod edit -require=github.com/joakimcarlsson/ai/model@v0.2.0 && go mod tidy
# ...repeat for each direct consumer, then commit + PR
```

After the PR merges, tag each cascaded module with a patch bump. Without
the cascade, users running `go get llm/gemini@latest` get the old `model`
version via MVS and the new constants don't resolve in their build.

Skip this step when the changed module isn't a shared dep (e.g. a fix
internal to `llm/anthropic`), or when only the module's own go.mod
changed (indirect dep bumps, `tidy` cleanup). Those have no
consumer-visible effect and don't need tags at all.

### 4. Warm the Go module proxy

```bash
scripts/release.sh warm -t llm/openai/v0.1.0
```

This ensures the tagged version is immediately available via `go get`.
Run it for every tag created in steps 2 and 3.

### 5. Create a dated GitHub Release

```bash
# Dry-run (shows what would be published)
scripts/release.sh release

# Publish
make release-publish
```

This creates a `release-YYYY-MM-DD` tag and a GitHub Release listing all
module tags created since the previous release.

## License

See [LICENSE](LICENSE) file.
