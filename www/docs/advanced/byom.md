# BYOM (Bring Your Own Model)

Use Ollama, LocalAI, vLLM, LM Studio, or any OpenAI-compatible inference server.

## Setup

```go
import (
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
)

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

## Custom Model Options

`model.NewCustomModel()` accepts options to describe your model's capabilities:

```go
customModel := model.NewCustomModel(
    model.WithModelID("my-model"),             // Unique identifier
    model.WithAPIModel("my-model-v1"),         // Model ID sent in API requests
    model.WithName("My Custom Model"),         // Human-readable name
    model.WithProvider("my-provider"),          // Provider identifier
    model.WithContextWindow(131_072),          // Max input tokens
    model.WithDefaultMaxTokens(4096),          // Default max output tokens
    model.WithStructuredOutput(true),          // Supports JSON schema output
    model.WithAttachments(true),               // Supports image/file inputs
    model.WithReasoning(true),                 // Supports chain-of-thought
    model.WithImageGeneration(false),          // Can generate images
    model.WithCostPer1MIn(1.50),              // Input cost per 1M tokens (USD)
    model.WithCostPer1MOut(5.00),             // Output cost per 1M tokens (USD)
    model.WithCostPer1MInCached(0.15),        // Cached input cost per 1M tokens
    model.WithCostPer1MOutCached(2.50),       // Cached output cost per 1M tokens
)
```

| Option | Description | Default |
|--------|-------------|---------|
| `WithModelID(id)` | Unique identifier for referencing the model | `""` |
| `WithAPIModel(name)` | Model name sent in API requests | `""` |
| `WithName(name)` | Human-readable display name | `""` |
| `WithProvider(provider)` | Provider identifier | `"custom"` |
| `WithContextWindow(tokens)` | Maximum input context size | `0` |
| `WithDefaultMaxTokens(tokens)` | Recommended max output tokens | `0` |
| `WithStructuredOutput(bool)` | Enable structured JSON output | `false` |
| `WithAttachments(bool)` | Enable image/file inputs | `false` |
| `WithReasoning(bool)` | Enable chain-of-thought reasoning | `false` |
| `WithImageGeneration(bool)` | Enable image generation | `false` |
| `WithCostPer1MIn(cost)` | Input token cost for cost tracking | `0` |
| `WithCostPer1MOut(cost)` | Output token cost for cost tracking | `0` |
| `WithCostPer1MInCached(cost)` | Cached input token cost | `0` |
| `WithCostPer1MOutCached(cost)` | Cached output token cost | `0` |

Setting these correctly enables the SDK to use features like structured output, context management strategies, and cost tracking with your custom model.

## Provider Configuration

`CustomProviderConfig` controls how the SDK connects to your server:

```go
provider := llm.RegisterCustomProvider("my-provider", llm.CustomProviderConfig{
    BaseURL:      "http://localhost:8080/v1",
    DefaultModel: customModel,
    ExtraHeaders: map[string]string{
        "X-API-Tenant": "my-tenant",
        "Authorization": "Bearer my-token",
    },
})
```

| Field | Description |
|-------|-------------|
| `BaseURL` | API endpoint URL (must serve OpenAI-compatible `/chat/completions`) |
| `DefaultModel` | Model configuration used when `WithModel` is not specified |
| `ExtraHeaders` | Additional HTTP headers for auth, routing, or custom metadata |

## Streaming

Custom providers support streaming out of the box:

```go
stream := client.StreamResponse(ctx, messages, nil)

for event := range stream {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventComplete:
        fmt.Println()
    case types.EventError:
        log.Fatal(event.Error)
    }
}
```

## Supported Servers

Any server that implements the OpenAI-compatible API:

- **Ollama** — `http://localhost:11434/v1`
- **LocalAI** — `http://localhost:8080/v1`
- **vLLM** — `http://localhost:8000/v1`
- **LM Studio** — `http://localhost:1234/v1`

See `example/byom/main.go` for a complete example.
