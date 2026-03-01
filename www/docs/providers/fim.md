# Fill-in-the-Middle (FIM)

Code completion by providing a prompt (code before the cursor) and an optional suffix (code after the cursor), with the model filling in the middle. Useful for code editors and IDE integrations.

## Supported Providers

| Provider | Model |
|----------|-------|
| Mistral | Codestral |
| DeepSeek | DeepSeek Coder |

## Setup

```go
import (
    "github.com/joakimcarlsson/ai/fim"
    "github.com/joakimcarlsson/ai/model"
)

client, err := fim.NewFIM(model.ProviderMistral,
    fim.WithAPIKey(os.Getenv("MISTRAL_API_KEY")),
    fim.WithModel(model.MistralModels[model.Codestral]),
)
if err != nil {
    log.Fatal(err)
}
```

## Basic Completion

```go
maxTokens := int64(100)

resp, err := client.Complete(ctx, fim.FIMRequest{
    Prompt:    "func Add(a, b int) int {\n    ",
    Suffix:    "\n}",
    MaxTokens: &maxTokens,
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Content)
// "return a + b"
```

## Streaming

```go
events := client.CompleteStream(ctx, fim.FIMRequest{
    Prompt:    "func Max(numbers []int) int {\n    ",
    Suffix:    "\n}",
    MaxTokens: &maxTokens,
})

for event := range events {
    switch event.Type {
    case fim.EventContentDelta:
        fmt.Print(event.Content)
    case fim.EventComplete:
        fmt.Printf("\nTokens: %d in, %d out\n",
            event.Response.Usage.InputTokens,
            event.Response.Usage.OutputTokens,
        )
    case fim.EventError:
        log.Fatal(event.Error)
    }
}
```

## FIMRequest

| Field | Type | Description |
|-------|------|-------------|
| `Prompt` | `string` | Code before the cursor (required) |
| `Suffix` | `string` | Code after the cursor (optional) |
| `MaxTokens` | `*int64` | Max tokens to generate |
| `Temperature` | `*float64` | Sampling temperature (0.0–1.0) |
| `TopP` | `*float64` | Nucleus sampling probability |
| `Stop` | `[]string` | Sequences that halt generation |
| `RandomSeed` | `*int64` | Seed for deterministic output |

## Client Options

| Option | Description |
|--------|-------------|
| `fim.WithAPIKey(key)` | API key for authentication |
| `fim.WithModel(m)` | Model to use |
| `fim.WithMaxTokens(n)` | Default max tokens |
| `fim.WithTemperature(t)` | Default temperature |
| `fim.WithTopP(p)` | Default top-p |
| `fim.WithTimeout(d)` | API request timeout |
| `fim.WithMistralOptions(...)` | Mistral-specific options |
| `fim.WithDeepSeekOptions(...)` | DeepSeek-specific options |
