# Reasoning / Extended Thinking

Models that support chain-of-thought reasoning can be configured to control
reasoning depth. Some providers also expose the model's thinking process via
`EventThinkingDelta` streaming events.

## Reasoning effort

Each LLM vendor module exports its own `WithReasoningEffort` (or
`WithThinkingLevel`) helper.

=== "OpenAI"

    ```go
    import llmopenai "github.com/joakimcarlsson/ai/llm/openai"

    client := llmopenai.NewLLM(
        llmopenai.WithAPIKey("your-key"),
        llmopenai.WithModel(model.OpenAIModels[model.O4Mini]),
        llmopenai.WithMaxTokens(16000),
        llmopenai.WithReasoningEffort(llmopenai.ReasoningEffortHigh),
    )
    ```

    | Level | Constant |
    |---|---|
    | Low | `llmopenai.ReasoningEffortLow` |
    | Medium | `llmopenai.ReasoningEffortMedium` |
    | High | `llmopenai.ReasoningEffortHigh` |

    OpenAI's Chat Completions API does not expose thinking content. The
    model reasons internally but `EventThinkingDelta` events are not emitted.

=== "Anthropic"

    ```go
    import llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"

    client := llmanthropic.NewLLM(
        llmanthropic.WithAPIKey("your-key"),
        llmanthropic.WithModel(model.AnthropicModels[model.Claude45Sonnet]),
        llmanthropic.WithMaxTokens(16000),
        llmanthropic.WithReasoningEffort(llmanthropic.ReasoningEffortHigh),
    )
    ```

    | Level | Constant |
    |---|---|
    | Low | `llmanthropic.ReasoningEffortLow` |
    | Medium | `llmanthropic.ReasoningEffortMedium` |
    | High | `llmanthropic.ReasoningEffortHigh` |
    | Max | `llmanthropic.ReasoningEffortMax` |

=== "Gemini"

    ```go
    import llmgemini "github.com/joakimcarlsson/ai/llm/gemini"

    client := llmgemini.NewLLM(
        llmgemini.WithAPIKey("your-key"),
        llmgemini.WithModel(model.GeminiModels[model.Gemini3Pro]),
        llmgemini.WithMaxTokens(16000),
        llmgemini.WithThinkingLevel(llmgemini.ThinkingLevelHigh),
    )
    ```

    | Level | Constant |
    |---|---|
    | Minimal | `llmgemini.ThinkingLevelMinimal` |
    | Low | `llmgemini.ThinkingLevelLow` |
    | Medium | `llmgemini.ThinkingLevelMedium` |
    | High | `llmgemini.ThinkingLevelHigh` |

## Streaming thinking events

Anthropic, Gemini, and OpenAI-compatible providers (Ollama, vLLM, etc.) that
expose `reasoning` deltas stream thinking content via `EventThinkingDelta`:

```go
import "github.com/joakimcarlsson/ai/types"

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

## OpenAI-compatible providers (Ollama, vLLM)

Reasoning models served via OpenAI-compatible APIs (Qwen, DeepSeek, etc.)
stream thinking content over the same `reasoning` delta channel. Use
`llm/openai` with a custom base URL and a custom model:

```go
import (
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/model"
)

ollama := llmopenai.NewLLM(
    llmopenai.WithAPIKey("ollama"),
    llmopenai.WithBaseURL("http://localhost:11434/v1"),
    llmopenai.WithModel(model.Model{
        ID:               "qwen3:14b",
        Name:             "Qwen3 14B",
        APIModel:         "qwen3:14b",
        Provider:         model.ProviderOpenAI,
        ContextWindow:    32768,
        DefaultMaxTokens: 4096,
        CanReason:        true,
    }),
    llmopenai.WithMaxTokens(4096),
)
```
