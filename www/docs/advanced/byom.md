# BYOM (Bring Your Own Model)

Use Ollama, LocalAI, vLLM, LM Studio, or any OpenAI-compatible inference
server. After the modular refactor, BYOM is just `llm/openai` with a custom
base URL — no separate code path. The `llm` module also ships a tiny config
registry for organising multiple BYOM endpoints.

## Direct setup

```go
import (
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/model"
)

llamaModel := model.NewCustomModel(
    model.WithModelID("llama3.2"),
    model.WithAPIModel("llama3.2:latest"),
    model.WithContextWindow(128_000),
)

client := llmopenai.NewLLM(
    llmopenai.WithBaseURL("http://localhost:11434/v1"),
    llmopenai.WithModel(llamaModel),
    llmopenai.WithMaxTokens(2000),
)

response, _ := client.SendMessages(ctx, messages, nil)
```

That's the whole story for a single endpoint.

## Registry helper (multiple endpoints)

When you've got several BYOM configurations and want to pass them around as
opaque IDs rather than re-typing URLs, use the `llm` module's registry:

```go
import "github.com/joakimcarlsson/ai/llm"

ollama := llm.RegisterCustomProvider("ollama", llm.CustomProviderConfig{
    BaseURL:      "http://localhost:11434/v1",
    DefaultModel: llamaModel,
})

// Later, in some other part of your code:
cfg, ok := llm.GetCustomProvider(ollama)
if !ok {
    log.Fatal("unknown provider")
}

client := llmopenai.NewLLM(
    llmopenai.WithBaseURL(cfg.BaseURL),
    llmopenai.WithExtraHeaders(cfg.ExtraHeaders),
    llmopenai.WithModel(cfg.DefaultModel),
)
```

The registry stores config; you still construct the client explicitly with
`llmopenai.NewLLM(...)`. There's no implicit factory that dispatches based
on provider ID — the modular refactor removed that, so callers know exactly
which vendor module they're invoking.

## Custom model options

```go
customModel := model.NewCustomModel(
    model.WithModelID("my-model"),
    model.WithAPIModel("my-model-v1"),         // sent in API requests
    model.WithName("My Custom Model"),         // human-readable
    model.WithProvider("my-provider"),         // provider identifier
    model.WithContextWindow(131_072),
    model.WithDefaultMaxTokens(4096),
    model.WithStructuredOutput(true),
    model.WithAttachments(true),
    model.WithReasoning(true),
    model.WithImageGeneration(false),
    model.WithCostPer1MIn(1.50),
    model.WithCostPer1MOut(5.00),
    model.WithCostPer1MInCached(0.15),
    model.WithCostPer1MOutCached(2.50),
)
```

| Option | Description | Default |
|---|---|---|
| `WithModelID(id)` | Unique identifier | `""` |
| `WithAPIModel(name)` | Model name sent in API requests | `""` |
| `WithName(name)` | Human-readable display name | `""` |
| `WithProvider(provider)` | Provider identifier | `"custom"` |
| `WithContextWindow(tokens)` | Max input context size | `0` |
| `WithDefaultMaxTokens(tokens)` | Recommended max output tokens | `0` |
| `WithStructuredOutput(bool)` | Enable structured JSON output | `false` |
| `WithAttachments(bool)` | Enable image/file inputs | `false` |
| `WithReasoning(bool)` | Enable chain-of-thought | `false` |
| `WithImageGeneration(bool)` | Enable image generation | `false` |
| `WithCostPer1MIn(cost)` | Input token cost per million | `0` |
| `WithCostPer1MOut(cost)` | Output token cost per million | `0` |
| `WithCostPer1MInCached(cost)` | Cached input token cost | `0` |
| `WithCostPer1MOutCached(cost)` | Cached output token cost | `0` |

Setting these correctly enables features like structured output, context
strategies, and cost tracking against your custom model.

## Extra headers

For tenant headers, custom auth, etc.:

```go
client := llmopenai.NewLLM(
    llmopenai.WithBaseURL("https://my-service.com/v1"),
    llmopenai.WithExtraHeaders(map[string]string{
        "X-API-Tenant":  "my-tenant",
        "Authorization": "Bearer my-token",
    }),
    llmopenai.WithModel(customModel),
)
```

## Streaming

Streaming works the same as any other vendor:

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

## Supported servers

Any server that implements the OpenAI-compatible `/chat/completions` API:

- **Ollama** — `http://localhost:11434/v1`
- **LocalAI** — `http://localhost:8080/v1`
- **vLLM** — `http://localhost:8000/v1`
- **LM Studio** — `http://localhost:1234/v1`
- **Groq** — `https://api.groq.com/openai/v1` (cloud, OpenAI-compatible)
- **OpenRouter** — `https://openrouter.ai/api/v1` (cloud, multi-vendor proxy)
- **xAI** — `https://api.x.ai/v1`
- **Mistral** — `https://api.mistral.ai/v1`
