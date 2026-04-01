# Batch Processing

Process bulk LLM and embedding requests efficiently using provider-native batch APIs or bounded concurrent execution.

## Native Batch APIs

Native batch APIs submit all requests as a single job that processes asynchronously on the provider side. Providers may offer reduced pricing for batch workloads (see the provider support table below for details). Results are typically returned within 24 hours, often much faster.

### OpenAI

```go
import (
    "github.com/joakimcarlsson/ai/batch"
    "github.com/joakimcarlsson/ai/model"
)

proc, _ := batch.New(
    model.ProviderOpenAI,
    batch.WithAPIKey("your-api-key"),
    batch.WithModel(model.OpenAIModels[model.GPT4o]),
    batch.WithPollInterval(30 * time.Second),
)

requests := []batch.Request{
    {
        ID:   "q1",
        Type: batch.RequestTypeChat,
        Messages: []message.Message{
            message.NewUserMessage("What is the capital of France?"),
        },
    },
    {
        ID:   "q2",
        Type: batch.RequestTypeChat,
        Messages: []message.Message{
            message.NewUserMessage("What is the capital of Japan?"),
        },
    },
}

resp, err := proc.Process(ctx, requests)
for _, r := range resp.Results {
    if r.Err != nil {
        fmt.Printf("[%s] Error: %v\n", r.ID, r.Err)
        continue
    }
    fmt.Printf("[%s] %s\n", r.ID, r.ChatResponse.Content)
}
```

### Anthropic

```go
proc, _ := batch.New(
    model.ProviderAnthropic,
    batch.WithAPIKey("your-api-key"),
    batch.WithModel(model.AnthropicModels[model.Claude4Sonnet]),
    batch.WithMaxTokens(1024),
    batch.WithPollInterval(30 * time.Second),
)
```

### Gemini / Vertex AI

```go
proc, _ := batch.New(
    model.ProviderGemini,
    batch.WithAPIKey("your-api-key"),
    batch.WithModel(model.GeminiModels[model.Gemini25Flash]),
    batch.WithPollInterval(30 * time.Second),
)
```

## Concurrent Fallback

For providers without native batch APIs, pass an existing LLM client. Requests run concurrently with a configurable concurrency limit.

```go
client, _ := llm.NewLLM(model.ProviderGroq,
    llm.WithAPIKey("your-api-key"),
    llm.WithModel(model.GroqModels[model.Llama4Scout]),
)

proc, _ := batch.New(
    model.ProviderGroq,
    batch.WithLLM(client),
    batch.WithMaxConcurrency(10),
)

resp, _ := proc.Process(ctx, requests)
```

## Batch Embeddings

```go
embedder, _ := embeddings.NewEmbedding(model.ProviderVoyage,
    embeddings.WithAPIKey("your-api-key"),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
)

proc, _ := batch.New(
    model.ProviderVoyage,
    batch.WithEmbedding(embedder),
    batch.WithMaxConcurrency(5),
)

requests := []batch.Request{
    {ID: "doc1", Type: batch.RequestTypeEmbedding, Texts: []string{"first document"}},
    {ID: "doc2", Type: batch.RequestTypeEmbedding, Texts: []string{"second document"}},
}

resp, _ := proc.Process(ctx, requests)
```

## Provider Support

| Provider | Native Batch | Discount (as of writing) | Supported Endpoints |
|----------|-------------|--------------------------|---------------------|
| OpenAI | ✅ | 50% | Chat, Embeddings |
| Anthropic | ✅ | 50% | Messages |
| Gemini | ✅ | 50% | Content, Embeddings |
| Vertex AI | ✅ | ~50% | Content, Embeddings |
| All others | Concurrent fallback | — | Chat, Embeddings |

## Progress Tracking

### Callback

```go
proc, _ := batch.New(
    model.ProviderOpenAI,
    batch.WithAPIKey("your-api-key"),
    batch.WithModel(model.OpenAIModels[model.GPT4o]),
    batch.WithProgressCallback(func(p batch.Progress) {
        fmt.Printf("%d/%d completed, %d failed [%s]\n",
            p.Completed, p.Total, p.Failed, p.Status)
    }),
)
```

### Async Channel

```go
ch, err := proc.ProcessAsync(ctx, requests)

for event := range ch {
    switch event.Type {
    case batch.EventItem:
        fmt.Printf("[%s] done\n", event.Result.ID)
    case batch.EventProgress:
        fmt.Printf("%d/%d\n", event.Progress.Completed, event.Progress.Total)
    case batch.EventComplete:
        fmt.Println("all done")
    case batch.EventError:
        fmt.Printf("batch error: %v\n", event.Err)
    }
}
```

## Error Handling

Individual request failures never fail the batch. Each result carries its own error.

```go
resp, err := proc.Process(ctx, requests)

for _, r := range resp.Results {
    if r.Err != nil {
        continue
    }
    // use r.ChatResponse or r.EmbedResponse
}

fmt.Printf("Completed: %d, Failed: %d\n", resp.Completed, resp.Failed)
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithAPIKey(key)` | API key for native batch providers | — |
| `WithModel(model)` | LLM model for chat batch requests | — |
| `WithEmbeddingModel(model)` | Embedding model for embedding batch requests | — |
| `WithMaxTokens(n)` | Max tokens per request | 4096 |
| `WithLLM(client)` | Existing LLM client for concurrent fallback | — |
| `WithEmbedding(client)` | Existing embedding client for concurrent fallback | — |
| `WithMaxConcurrency(n)` | Max parallel requests in concurrent mode | 10 |
| `WithProgressCallback(fn)` | Progress update callback | — |
| `WithPollInterval(d)` | Polling interval for native batch APIs | 30s |
| `WithTimeout(d)` | Request timeout | — |
| `WithOpenAIOptions(...)` | OpenAI-specific options (base URL, headers) | — |
| `WithGeminiOptions(...)` | Gemini-specific options (backend) | — |
