# Quick Start

## Basic chat

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
)

func main() {
    ctx := context.Background()

    client := llmopenai.NewLLM(
        llmopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
        llmopenai.WithMaxTokens(1000),
    )

    response, err := client.SendMessages(ctx, []message.Message{
        message.NewUserMessage("Hello, how are you?"),
    }, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

The `client` value satisfies the `llm.LLM` interface, so swapping vendors is
just a matter of changing which package you construct from. The rest of your
code (`SendMessages`, `StreamResponse`, etc.) stays the same.

## Streaming

```go
import "github.com/joakimcarlsson/ai/types"

stream := client.StreamResponse(ctx, messages, nil)

for event := range stream {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventComplete:
        fmt.Printf("\nTokens: %d in / %d out\n",
            event.Response.Usage.InputTokens,
            event.Response.Usage.OutputTokens)
    case types.EventError:
        log.Fatal(event.Error)
    }
}
```

## Multimodal (images)

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

response, err := client.SendMessages(ctx, []message.Message{msg}, nil)
```

## Switching vendors

Anthropic instead of OpenAI:

```go
import llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"

client := llmanthropic.NewLLM(
    llmanthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    llmanthropic.WithModel(model.AnthropicModels[model.Claude45Sonnet]),
    llmanthropic.WithMaxTokens(1000),
)
```

The `client` is still a `llm.LLM`, so the rest of the program is unchanged.

## Your first agent

The `agent` module wires an LLM client, tools, session storage, and optional
memory into a runtime:

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/session"
)

myAgent := agent.New(client,
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithTools(&weatherTool{}),
    agent.WithSession("user-123", session.FileStore("./sessions")),
)

response, _ := myAgent.Chat(ctx, "What's the weather in Tokyo?")
fmt.Println(response.Content)
```

See the [Agent Framework](../agent/overview.md) section for the full guide.
