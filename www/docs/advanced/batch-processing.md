# Batch Processing

Process bulk LLM and embedding requests efficiently using provider-native
batch APIs or bounded concurrent execution. Each batch backend is its own Go
module under `batch/`.

## Native batch APIs

Native batch APIs submit all requests as a single async job. Providers may
offer reduced pricing for batch workloads. Results return within 24 hours,
often much faster.

### OpenAI

```go
import (
    "github.com/joakimcarlsson/ai/batch"
    batchopenai "github.com/joakimcarlsson/ai/batch/openai"
    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
)

proc := batchopenai.NewProcessor(
    batchopenai.WithAPIKey("your-api-key"),
    batchopenai.WithModel(model.OpenAIModels[model.GPT4o]),
    batchopenai.WithPollInterval(30 * time.Second),
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
import batchanthropic "github.com/joakimcarlsson/ai/batch/anthropic"

proc := batchanthropic.NewProcessor(
    batchanthropic.WithAPIKey("your-api-key"),
    batchanthropic.WithModel(model.AnthropicModels[model.Claude45Sonnet]),
    batchanthropic.WithMaxTokens(1024),
    batchanthropic.WithPollInterval(30 * time.Second),
)
```

Anthropic's batch API supports chat only — submitting an embedding request
returns an error.

### Gemini / Vertex AI

```go
import batchgemini "github.com/joakimcarlsson/ai/batch/gemini"

proc := batchgemini.NewProcessor(
    batchgemini.WithAPIKey("your-api-key"),
    batchgemini.WithModel(model.GeminiModels[model.Gemini25Flash]),
    batchgemini.WithPollInterval(30 * time.Second),
)
```

For Vertex AI:

```go
import "google.golang.org/genai"

proc := batchgemini.NewProcessor(
    batchgemini.WithModel(...),
    batchgemini.WithBackend(genai.BackendVertexAI),
)
```

## Concurrent fallback

For providers without a native batch API, pass an existing LLM and/or
embedding client to the concurrent runner:

```go
import (
    batchconcurrent "github.com/joakimcarlsson/ai/batch/concurrent"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

groq := llmopenai.NewLLM(
    llmopenai.WithAPIKey(os.Getenv("GROQ_API_KEY")),
    llmopenai.WithBaseURL("https://api.groq.com/openai/v1"),
    llmopenai.WithModel(model.GroqModels[model.Llama4Scout]),
)

proc := batchconcurrent.NewProcessor(
    batchconcurrent.WithLLM(groq),
    batchconcurrent.WithMaxConcurrency(10),
)

resp, _ := proc.Process(ctx, requests)
```

The concurrent runner is no longer an automatic fallthrough — you pick it
explicitly. This means consumers know exactly which package they're getting
and which deps it pulls.

## Batch embeddings

```go
import (
    embvoyage "github.com/joakimcarlsson/ai/embeddings/voyage"
)

embedder := embvoyage.NewEmbedding(
    embvoyage.WithAPIKey(os.Getenv("VOYAGE_API_KEY")),
    embvoyage.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
)

proc := batchconcurrent.NewProcessor(
    batchconcurrent.WithEmbedding(embedder),
    batchconcurrent.WithMaxConcurrency(5),
)

requests := []batch.Request{
    {ID: "doc1", Type: batch.RequestTypeEmbedding, Texts: []string{"first document"}},
    {ID: "doc2", Type: batch.RequestTypeEmbedding, Texts: []string{"second document"}},
}

resp, _ := proc.Process(ctx, requests)
```

## Provider support

| Module | Native batch | Discount (as of writing) | Supported endpoints |
|---|---|---|---|
| `batch/openai` | ✅ | 50% | Chat, Embeddings |
| `batch/anthropic` | ✅ | 50% | Messages (chat only) |
| `batch/gemini` | ✅ | 50% | Content, Embeddings |
| `batch/concurrent` | (concurrent fallback) | — | Chat, Embeddings via existing client |

## Progress tracking

### Callback

```go
proc := batchopenai.NewProcessor(
    batchopenai.WithAPIKey("your-api-key"),
    batchopenai.WithModel(model.OpenAIModels[model.GPT4o]),
    batchopenai.WithProgressCallback(func(p batch.Progress) {
        fmt.Printf("%d/%d completed, %d failed [%s]\n",
            p.Completed, p.Total, p.Failed, p.Status)
    }),
)
```

### Async event channel

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

## Error handling

Individual request failures never fail the batch — each result carries its
own error:

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

## Common options

Across the native-batch processors:

```go
batchopenai.WithAPIKey("...")
batchopenai.WithModel(...)
batchopenai.WithEmbeddingModel(...)
batchopenai.WithMaxTokens(4096)
batchopenai.WithProgressCallback(fn)
batchopenai.WithPollInterval(30 * time.Second)
batchopenai.WithTimeout(60 * time.Second)
```

OpenAI-specific:

```go
batchopenai.WithBaseURL("https://custom-endpoint")
batchopenai.WithExtraHeaders(map[string]string{"X-Header": "value"})
```

Gemini-specific:

```go
batchgemini.WithBackend(genai.BackendVertexAI)
```

Concurrent fallback:

```go
batchconcurrent.WithLLM(client)
batchconcurrent.WithEmbedding(client)
batchconcurrent.WithMaxConcurrency(10)
batchconcurrent.WithProgressCallback(fn)
```
