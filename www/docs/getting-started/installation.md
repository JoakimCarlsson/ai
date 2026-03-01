# Installation

## Requirements

- Go 1.24 or later

## Install

```bash
go get github.com/joakimcarlsson/ai
```

## Import

```go
import (
    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
)
```

## Provider API Keys

Each provider requires its own API key. Set them as environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."
export VOYAGE_API_KEY="..."
export XAI_API_KEY="..."
export ELEVENLABS_API_KEY="..."
```

Or pass them directly when creating a client:

```go
client, err := llm.NewLLM(
    model.ProviderOpenAI,
    llm.WithAPIKey("your-api-key"),
    llm.WithModel(model.OpenAIModels[model.GPT4o]),
)
```
