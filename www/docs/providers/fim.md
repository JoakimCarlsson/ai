# Fill-in-the-Middle (FIM)

Code completion that takes a `Prompt` (code before the cursor) and an optional
`Suffix` (code after the cursor) and fills in the middle. Useful for code
editors and IDE integrations. The `fim` modality lives at `fim/`; vendors
under `fim/<name>/`.

## Mistral Codestral

```go
import (
    "github.com/joakimcarlsson/ai/fim"
    fimmistral "github.com/joakimcarlsson/ai/fim/mistral"
    "github.com/joakimcarlsson/ai/model"
)

client := fimmistral.NewFIM(
    fimmistral.WithAPIKey(os.Getenv("MISTRAL_API_KEY")),
    fimmistral.WithModel(model.MistralModels[model.Codestral]),
)

resp, err := client.Complete(ctx, fim.Request{
    Prompt: "def add(a, b):\n    ",
    Suffix: "\n    return result",
})
fmt.Println(resp.Content)
```

## DeepSeek

```go
import fimdeepseek "github.com/joakimcarlsson/ai/fim/deepseek"

client := fimdeepseek.NewFIM(
    fimdeepseek.WithAPIKey(os.Getenv("DEEPSEEK_API_KEY")),
    fimdeepseek.WithModel(model.DeepSeekModels[model.DeepSeekCoder]),
)
```

## Streaming

```go
events := client.CompleteStream(ctx, fim.Request{
    Prompt: prompt,
    Suffix: suffix,
})

for event := range events {
    switch event.Type {
    case fim.EventContentDelta:
        fmt.Print(event.Content)
    case fim.EventComplete:
        fmt.Printf("\nFinish: %s\n", event.Response.FinishReason)
    case fim.EventError:
        log.Fatal(event.Error)
    }
}
```

## Per-request overrides

The `fim.Request` struct lets you override constructor defaults per call:

```go
maxTokens := int64(500)
temp := 0.2

resp, err := client.Complete(ctx, fim.Request{
    Prompt:      prompt,
    Suffix:      suffix,
    MaxTokens:   &maxTokens,
    Temperature: &temp,
    Stop:        []string{"\n\n"},
})
```

## Vendor-specific options

Mistral:

```go
fimmistral.WithMinTokens(20)
```

DeepSeek:

```go
fimdeepseek.WithFrequencyPenalty(0.3)
fimdeepseek.WithPresencePenalty(0.3)
fimdeepseek.WithEcho(true)
```
