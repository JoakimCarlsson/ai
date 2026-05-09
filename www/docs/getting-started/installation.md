# Installation

## Requirements

- Go 1.25 or later

## Install only what you use

Each modality and vendor is a separate Go module. Pull just the ones you need.

### Common combinations

**LLM with OpenAI:**

```bash
go get github.com/joakimcarlsson/ai/llm
go get github.com/joakimcarlsson/ai/llm/openai
go get github.com/joakimcarlsson/ai/message
go get github.com/joakimcarlsson/ai/model
```

**LLM with Anthropic:**

```bash
go get github.com/joakimcarlsson/ai/llm
go get github.com/joakimcarlsson/ai/llm/anthropic
go get github.com/joakimcarlsson/ai/message
go get github.com/joakimcarlsson/ai/model
```

**Embeddings (Voyage) + LLM (OpenAI) + agent runtime:**

```bash
go get github.com/joakimcarlsson/ai/agent
go get github.com/joakimcarlsson/ai/llm
go get github.com/joakimcarlsson/ai/llm/openai
go get github.com/joakimcarlsson/ai/embeddings
go get github.com/joakimcarlsson/ai/embeddings/voyage
go get github.com/joakimcarlsson/ai/message
go get github.com/joakimcarlsson/ai/model
```

**Persistent memory with pgvector:**

```bash
go get github.com/joakimcarlsson/ai/agent
go get github.com/joakimcarlsson/ai/memory
go get github.com/joakimcarlsson/ai/memory/pgvector
go get github.com/joakimcarlsson/ai/embeddings
go get github.com/joakimcarlsson/ai/embeddings/openai
```

## Import shape

The Go convention is to alias the vendor package with a short name to avoid
clashing with the modality package:

```go
import (
    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)
```

For agent-runtime code that wires multiple modalities:

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/memory"
    pgvectormem "github.com/joakimcarlsson/ai/memory/pgvector"
    embopenai "github.com/joakimcarlsson/ai/embeddings/openai"
    llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
    "github.com/joakimcarlsson/ai/model"
)
```

## API keys

Each vendor needs its own credential. Common environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GEMINI_API_KEY="..."
export VOYAGE_API_KEY="pa-..."
export COHERE_API_KEY="..."
export ELEVENLABS_API_KEY="..."
export DEEPGRAM_API_KEY="..."
export ASSEMBLYAI_API_KEY="..."
```

You can also pass the key directly to the constructor:

```go
client := llmopenai.NewLLM(
    llmopenai.WithAPIKey("sk-..."),
    llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
)
```

## OpenAI-compatible providers

Groq, OpenRouter, xAI, Mistral, DeepSeek, Perplexity, and any
OpenAI-compatible endpoint use `llm/openai` with a custom base URL — no
separate vendor module:

```go
client := llmopenai.NewLLM(
    llmopenai.WithAPIKey(os.Getenv("GROQ_API_KEY")),
    llmopenai.WithBaseURL("https://api.groq.com/openai/v1"),
    llmopenai.WithModel(model.GroqModels[model.LLaMA3_70B]),
)
```

See [BYOM](../advanced/byom.md) for the registry helper that organises these.
