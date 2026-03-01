# LLM Providers

## Creating a Client

```go
import (
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
)

client, err := llm.NewLLM(
    model.ProviderOpenAI,
    llm.WithAPIKey("your-api-key"),
    llm.WithModel(model.OpenAIModels[model.GPT4o]),
    llm.WithMaxTokens(1000),
)
```

## Sending Messages

```go
messages := []message.Message{
    message.NewUserMessage("Hello, how are you?"),
}

response, err := client.SendMessages(ctx, messages, nil)
fmt.Println(response.Content)
```

## Streaming

```go
stream := client.StreamResponse(ctx, messages, nil)

for event := range stream {
    switch event.Type {
    case types.EventTypeContentDelta:
        fmt.Print(event.Content)
    case types.EventTypeFinal:
        fmt.Printf("\nTokens used: %d\n", event.Response.Usage.InputTokens)
    case types.EventTypeError:
        log.Fatal(event.Error)
    }
}
```

## Multimodal (Images)

```go
imageData, err := os.ReadFile("image.png")
if err != nil {
    log.Fatal(err)
}

msg := message.NewUserMessage("What's in this image?")
msg.AddAttachment(message.Attachment{
    MIMEType: "image/png",
    Data:     imageData,
})

messages := []message.Message{msg}
response, err := client.SendMessages(ctx, messages, nil)
```

## Client Options

```go
client, err := llm.NewLLM(
    model.ProviderOpenAI,
    llm.WithAPIKey("your-key"),
    llm.WithModel(model.OpenAIModels[model.GPT4o]),
    llm.WithMaxTokens(2000),
    llm.WithTemperature(0.7),
    llm.WithTopP(0.9),
    llm.WithTimeout(30*time.Second),
    llm.WithStopSequences("STOP", "END"),
)
```

## Provider-Specific Options

```go
// Anthropic
llm.WithAnthropicOptions(
    llm.WithAnthropicBeta("beta-feature"),
)

// OpenAI
llm.WithOpenAIOptions(
    llm.WithOpenAIBaseURL("custom-endpoint"),
    llm.WithOpenAIExtraHeaders(map[string]string{
        "Custom-Header": "value",
    }),
)
```
