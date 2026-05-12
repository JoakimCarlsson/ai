# Modules

The library is published as ~50 independent Go modules. Every module lives at
`github.com/joakimcarlsson/ai/<path>` and is versioned independently with
path-prefixed tags (e.g. `llm/openai/v0.1.0`).

You install only what you use. A typical OpenAI chat client needs `llm`,
`llm/openai`, `message`, and `model`.

## Tier 0 — Leaves

Dependency-free building blocks. No vendor SDKs.

| Module | Purpose |
|---|---|
| `model` | Model catalog, capability flags, custom-model builder |
| `message` | Conversation message types and constructors |
| `tool` | Tool interface, MCP integration, function-tool helpers |
| `schema` | JSON Schema generation for tool inputs and structured output |
| `tracing` | OpenTelemetry setup helper for traces, metrics, and logs |
| `prompt` | Prompt template rendering |
| `types` | Shared event types (`EventContentDelta`, `EventThinkingDelta`, etc.) |

## Tier 1 — Modality interfaces

Define the contract for each capability. No vendor SDKs; vendor modules
implement these.

| Module | Purpose |
|---|---|
| `llm` | Chat completion interface, retry config, tracing wrapper |
| `embeddings` | Text / multimodal / contextualized embedding interface |
| `tts` | Text-to-speech interface (with optional `ForcedAlignmentProvider`) |
| `stt` | Speech-to-text interface (transcribe + translate, streaming) |
| `image` | Image generation interface |
| `rerankers` | Document reranking interface |
| `fim` | Fill-in-the-middle code completion interface |

## Tier 2 — Vendor implementations

Each carries exactly one vendor SDK.

### LLM

| Module | Vendor SDK |
|---|---|
| `llm/openai` | `openai-go` — Chat Completions via `NewLLM` (also OpenRouter, xAI, Mistral via `WithBaseURL`); Responses API with server-side built-in tools via `NewResponsesLLM` |
| `llm/anthropic` | `anthropic-sdk-go` (also Bedrock backend); server-side `web_search` via `WithWebSearch` |
| `llm/gemini` | `google.golang.org/genai`; server-side `google_search`, `code_execution`, `url_context` via dedicated options |
| `llm/groq` | `openai-go` — fast OpenAI-compatible chat via `NewLLM`; compound-model built-ins (`browser_search`, `code_execution`, `visit_website`) via `NewCompoundLLM` |
| `llm/xai` | `openai-go` — OpenAI-compatible chat via `NewLLM`; Responses API built-ins (`web_search`, `x_search`, `code_execution`) via `NewResponsesLLM` |
| `llm/vertexai` | `google.golang.org/genai` (Vertex AI backend) |
| `llm/azure` | `openai-go` against Azure OpenAI |
| `llm/bedrock` | `aws-sdk-go-v2` Bedrock Runtime |

### Embeddings

| Module | Vendor SDK |
|---|---|
| `embeddings/openai` | `openai-go` |
| `embeddings/voyage` | `net/http` |
| `embeddings/cohere` | `net/http` |
| `embeddings/gemini` | `google.golang.org/genai` |
| `embeddings/mistral` | `net/http` |
| `embeddings/bedrock` | `aws-sdk-go-v2` Bedrock Runtime |

### TTS

| Module | Vendor SDK |
|---|---|
| `tts/openai` | `openai-go` |
| `tts/elevenlabs` | `net/http` |
| `tts/google` | `cloud.google.com/go/texttospeech` |
| `tts/azure` | `net/http` (Azure Cognitive Services Speech) |
| `tts/deepgram` | `net/http` |

### STT

| Module | Vendor SDK |
|---|---|
| `stt/openai` | `openai-go` |
| `stt/elevenlabs` | `net/http` + `gorilla/websocket` for streaming |
| `stt/deepgram` | `deepgram-go-sdk` |
| `stt/assemblyai` | `assemblyai-go-sdk` |
| `stt/google` | `cloud.google.com/go/speech` |

### Image generation

| Module | Vendor SDK |
|---|---|
| `image/openai` | `openai-go` (also xAI via `WithBaseURL`) |
| `image/gemini` | `google.golang.org/genai` (also Vertex AI) |

### Rerankers

| Module | Vendor SDK |
|---|---|
| `rerankers/voyage` | `net/http` |
| `rerankers/cohere` | `net/http` |

### Fill-in-the-middle

| Module | Vendor SDK |
|---|---|
| `fim/mistral` | `net/http` |
| `fim/deepseek` | `net/http` |

## Tier 3 — Utilities

| Module | Purpose |
|---|---|
| `tokens/sliding` | Sliding-window context strategy |
| `tokens/truncate` | Hard-truncate context strategy |
| `tokens/summarize` | Summarization-based context strategy |
| `batch` | Batch request / progress / event types |
| `batch/openai` | OpenAI native batch API processor |
| `batch/anthropic` | Anthropic native batch API processor |
| `batch/gemini` | Gemini / Vertex AI native batch API processor |
| `batch/concurrent` | Bounded-concurrency runner over any LLM / embedding client |

## Tier 4 — Agent runtimes and conversation primitives

| Module | Purpose |
|---|---|
| `agent` | Agent runtime: chat, streaming, hooks, tools, sub-agents, handoffs, fan-out |
| `voice` | Voice-first agent: streaming STT → LLM → TTS pipeline with tool calls |
| `memory` | Persistent memory interface, dedup + extraction helpers |
| `session` | Conversation session storage interfaces and implementations |

`agent/team` is a sub-package of `agent` (same module).

## Tier 5 — Persistence

| Module | Purpose |
|---|---|
| `memory/pgvector` | PostgreSQL + pgvector backend with HNSW vector search |
| `memory/postgres` | PostgreSQL session + memory store |
| `memory/sqlite` | SQLite session + memory store |

## Adding new modules

The repo is a Go workspace; each module has its own `go.mod`. To add a new
vendor implementation:

1. Create the directory and `go.mod` (declare module path
   `github.com/joakimcarlsson/ai/<modality>/<vendor>`).
2. Add a `replace` directive for any internal modules it imports.
3. Add the path to `go.work.example`.
4. Tag releases as `<modality>/<vendor>/vX.Y.Z`.

See [the release process](https://github.com/joakimcarlsson/ai#release-process)
for details on tagging and publishing.
