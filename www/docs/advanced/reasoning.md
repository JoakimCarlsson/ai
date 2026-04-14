# Reasoning / Extended Thinking

Models that support chain-of-thought reasoning can expose their internal thinking process via `EventThinkingDelta` streaming events.

## Provider Setup

=== "OpenAI"

    ```go
    client, err := llm.NewLLM(
        model.ProviderOpenAI,
        llm.WithAPIKey("your-key"),
        llm.WithModel(model.OpenAIModels[model.O4Mini]),
        llm.WithMaxTokens(16000),
        llm.WithOpenAIOptions(
            llm.WithReasoningEffort(llm.OpenAIReasoningEffortHigh),
        ),
    )
    ```

    | Level | Constant |
    |-------|----------|
    | Low | `OpenAIReasoningEffortLow` |
    | Medium | `OpenAIReasoningEffortMedium` |
    | High | `OpenAIReasoningEffortHigh` |

=== "Anthropic"

    ```go
    client, err := llm.NewLLM(
        model.ProviderAnthropic,
        llm.WithAPIKey("your-key"),
        llm.WithModel(model.AnthropicModels[model.Claude4Sonnet]),
        llm.WithMaxTokens(16000),
        llm.WithAnthropicOptions(
            llm.WithAnthropicReasoningEffort(llm.AnthropicReasoningEffortHigh),
        ),
    )
    ```

    | Level | Constant |
    |-------|----------|
    | Low | `AnthropicReasoningEffortLow` |
    | Medium | `AnthropicReasoningEffortMedium` |
    | High | `AnthropicReasoningEffortHigh` |
    | Max | `AnthropicReasoningEffortMax` |

=== "Gemini"

    ```go
    client, err := llm.NewLLM(
        model.ProviderGemini,
        llm.WithAPIKey("your-key"),
        llm.WithModel(model.GeminiModels[model.Gemini3Pro]),
        llm.WithMaxTokens(16000),
        llm.WithGeminiOptions(
            llm.WithGeminiThinkingLevel(llm.GeminiThinkingLevelHigh),
        ),
    )
    ```

    | Level | Constant |
    |-------|----------|
    | Minimal | `GeminiThinkingLevelMinimal` |
    | Low | `GeminiThinkingLevelLow` |
    | Medium | `GeminiThinkingLevelMedium` |
    | High | `GeminiThinkingLevelHigh` |

## Streaming Thinking Events

```go
for event := range client.StreamResponse(ctx, messages, nil) {
    switch event.Type {
    case types.EventThinkingDelta:
        fmt.Print(event.Thinking)
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventComplete:
        fmt.Printf("\nTokens: %d in, %d out\n",
            event.Response.Usage.InputTokens,
            event.Response.Usage.OutputTokens,
        )
    case types.EventError:
        log.Fatal(event.Error)
    }
}
```

The same pattern works with agents via `ChatStream`:

```go
for event := range myAgent.ChatStream(ctx, "Think about this carefully...") {
    switch event.Type {
    case types.EventThinkingDelta:
        fmt.Print(event.Thinking)
    case types.EventContentDelta:
        fmt.Print(event.Content)
    }
}
```
