# Batch Processing

Process bulk LLM and embedding requests efficiently using provider-native batch APIs (50% cost savings) or bounded concurrent execution.

## Concurrent Processing

Works with any provider. Sends requests concurrently with a configurable concurrency limit.

```go
import (
    "github.com/joakimcarlsson/ai/batch"
    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
)

client, _ := llm.NewLLM(model.ProviderOpenAI,
    llm.WithAPIKey("your-api-key"),
    llm.WithModel(model.OpenAIModels[model.GPT4o]),
)

proc := batch.New(
    batch.WithLLM(client),
    batch.WithMaxConcurrency(10),
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

## Batch Embeddings

```go
embedder, _ := embeddings.NewEmbedding(model.ProviderVoyage,
    embeddings.WithAPIKey("your-api-key"),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
)

proc := batch.New(
    batch.WithEmbedding(embedder),
    batch.WithMaxConcurrency(5),
)

requests := []batch.Request{
    {ID: "doc1", Type: batch.RequestTypeEmbedding, Texts: []string{"first document"}},
    {ID: "doc2", Type: batch.RequestTypeEmbedding, Texts: []string{"second document"}},
}

resp, _ := proc.Process(ctx, requests)
```

## Native Batch APIs

Native batch APIs submit all requests as a single job that processes asynchronously on the provider side. This gives 50% cost savings and higher rate limits, with a 24-hour turnaround (typically much faster).

### OpenAI

```go
import (
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

client := openai.NewClient(option.WithAPIKey("your-api-key"))

proc := batch.New(
    batch.WithOpenAIClient(client),
    batch.WithPollInterval(30 * time.Second),
)

resp, err := proc.Process(ctx, requests)
```

Also works with Mistral and Azure OpenAI (same batch format, different base URL).

### Anthropic

```go
import (
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

client := anthropic.NewClient(option.WithAPIKey("your-api-key"))

proc := batch.New(
    batch.WithAnthropicClient(client),
    batch.WithPollInterval(30 * time.Second),
)

resp, err := proc.Process(ctx, requests)
```

### Gemini / Vertex AI

```go
import "google.golang.org/genai"

client, _ := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  "your-api-key",
    Backend: genai.BackendGeminiAPI,
})

proc := batch.New(
    batch.WithGeminiClient(client, "gemini-2.5-flash"),
    batch.WithPollInterval(30 * time.Second),
)

resp, err := proc.Process(ctx, requests)
```

## Provider Support

| Provider | Native Batch | Cost Savings | Supported Endpoints |
|----------|-------------|-------------|---------------------|
| OpenAI | ✅ | 50% | Chat, Embeddings |
| Anthropic | ✅ | 50% | Messages |
| Gemini | ✅ | 50% | Content, Embeddings |
| Vertex AI | ✅ | ~50% | Content, Embeddings |
| Mistral | ✅ | 50% | Chat, Embeddings |
| Azure OpenAI | ✅ | 50% | Chat, Embeddings |
| All others | Concurrent fallback | — | Chat, Embeddings |

## Progress Tracking

### Callback

```go
proc := batch.New(
    batch.WithLLM(client),
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
// err is only non-nil for systemic failures (context cancelled, file upload failed)

for _, r := range resp.Results {
    if r.Err != nil {
        // this request failed, others may have succeeded
        continue
    }
    // use r.ChatResponse or r.EmbedResponse
}

fmt.Printf("Completed: %d, Failed: %d\n", resp.Completed, resp.Failed)
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithLLM(client)` | LLM client for concurrent chat requests | — |
| `WithEmbedding(client)` | Embedding client for concurrent embedding requests | — |
| `WithMaxConcurrency(n)` | Max parallel requests in concurrent mode | 10 |
| `WithProgressCallback(fn)` | Progress update callback | — |
| `WithPollInterval(d)` | Polling interval for native batch APIs | 30s |
| `WithOpenAIClient(client)` | Enable OpenAI native batch | — |
| `WithAnthropicClient(client)` | Enable Anthropic native batch | — |
| `WithGeminiClient(client, model)` | Enable Gemini native batch | — |
