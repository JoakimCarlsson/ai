# Reasoning / Extended Thinking

Models that support chain-of-thought reasoning can be configured to control reasoning depth. Some providers also expose the model's thinking process via `EventThinkingDelta` streaming events.

## Reasoning Effort

Controls how much effort the model spends on internal reasoning before generating a response.

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

    OpenAI's Chat Completions API does not expose thinking content. The model reasons internally but `EventThinkingDelta` events are not emitted.

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

Anthropic, Gemini, and OpenAI-compatible providers (Ollama, vLLM, etc.) stream thinking content via `EventThinkingDelta`:

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

## OpenAI-Compatible Providers

Providers like Ollama and vLLM that serve reasoning models (Qwen, DeepSeek, etc.) via an OpenAI-compatible API do stream thinking content. Use a custom model with `WithOpenAIBaseURL`:

```go
client, err := llm.NewLLM(
    model.ProviderOpenAI,
    llm.WithAPIKey("ollama"),
    llm.WithModel(model.Model{
        ID:               "qwen3:14b",
        Name:             "Qwen3 14B",
        APIModel:         "qwen3:14b",
        Provider:         model.ProviderOpenAI,
        ContextWindow:    32768,
        DefaultMaxTokens: 4096,
        CanReason:        true,
    }),
    llm.WithMaxTokens(4096),
    llm.WithOpenAIOptions(
        llm.WithOpenAIBaseURL("http://localhost:11434/v1"),
    ),
)
```
