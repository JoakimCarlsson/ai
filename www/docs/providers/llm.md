# LLM Providers

Each native LLM vendor is published as its own Go module under `llm/`. The
client returned from a vendor's `NewLLM(...)` satisfies the `llm.LLM`
interface, so once you've constructed it the rest of your code is
vendor-agnostic.

## Creating a client

```go
import (
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/model"
)

client := llmopenai.NewLLM(
    llmopenai.WithAPIKey("your-api-key"),
    llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
    llmopenai.WithMaxTokens(1000),
)
```

For Anthropic instead:

```go
import llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"

client := llmanthropic.NewLLM(
    llmanthropic.WithAPIKey("..."),
    llmanthropic.WithModel(model.AnthropicModels[model.Claude45Sonnet]),
    llmanthropic.WithMaxTokens(1000),
)
```

## Sending messages

```go
import "github.com/joakimcarlsson/ai/message"

response, err := client.SendMessages(ctx, []message.Message{
    message.NewUserMessage("Hello, how are you?"),
}, nil)
fmt.Println(response.Content)
```

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
imageData, _ := os.ReadFile("image.png")

msg := message.NewUserMessage("What's in this image?")
msg.AddAttachment(message.Attachment{
    MIMEType: "image/png",
    Data:     imageData,
})

response, err := client.SendMessages(ctx, []message.Message{msg}, nil)
```

## Common options

Every vendor exports the standard set:

```go
llmopenai.WithAPIKey("...")
llmopenai.WithModel(model.OpenAIModels[model.GPT4o])
llmopenai.WithMaxTokens(2000)
llmopenai.WithTemperature(0.7)
llmopenai.WithTopP(0.9)
llmopenai.WithTopK(40)
llmopenai.WithStopSequences("STOP", "END")
llmopenai.WithTimeout(30 * time.Second)
```

## Vendor-specific options

OpenAI:

```go
llmopenai.WithBaseURL("https://custom-endpoint")
llmopenai.WithExtraHeaders(map[string]string{"X-My-Header": "value"})
llmopenai.WithReasoningEffort(llmopenai.ReasoningEffortHigh)
llmopenai.WithFrequencyPenalty(0.5)
llmopenai.WithPresencePenalty(0.5)
llmopenai.WithSeed(42)
llmopenai.WithParallelToolCalls(false)
```

Anthropic:

```go
llmanthropic.WithBedrock(true)              // route through AWS Bedrock
llmanthropic.WithDisableCache()
llmanthropic.WithReasoningEffort(llmanthropic.ReasoningEffortHigh)
```

Gemini:

```go
import llmgemini "github.com/joakimcarlsson/ai/llm/gemini"

llmgemini.WithThinkingLevel(llmgemini.ThinkingLevelHigh)
llmgemini.WithFrequencyPenalty(0.5)
llmgemini.WithSeed(42)
```

## Provider built-in tools

Server-side built-in tools (web search, code execution, file search) run
inside the provider's infrastructure. They're opt-in per-client; results land
inline in `Response.Content`, with structured metadata under
`Response.ProviderMetadata`. See [Tool Calling](../advanced/tools.md#provider-built-in-tools)
for the full picture; below is the per-provider surface.

Anthropic — `web_search`:

```go
llmanthropic.WithWebSearch(llmanthropic.WebSearchConfig{
    MaxUses:        5,
    AllowedDomains: []string{"go.dev"},
    BlockedDomains: nil,
    UserLocation: &llmanthropic.WebSearchUserLocation{
        City: "Stockholm", Country: "SE",
    },
})
```

Gemini — `google_search`, `code_execution`, `url_context`:

```go
llmgemini.WithGoogleSearch()
llmgemini.WithCodeExecution()
llmgemini.WithURLContext()
```

OpenAI (Responses API) — `web_search`, `file_search`, `code_interpreter`. The
Responses API is a separate surface from Chat Completions; use
`NewResponsesLLM` instead of `NewLLM`:

```go
client := llmopenai.NewResponsesLLM(
    llmopenai.WithResponsesAPIKey(os.Getenv("OPENAI_API_KEY")),
    llmopenai.WithResponsesModel(model.OpenAIModels[model.GPT5]),
    llmopenai.WithResponsesMaxTokens(1024),
    llmopenai.WithWebSearch(llmopenai.WebSearchOpts{
        SearchContextSize: llmopenai.SearchContextMedium,
    }),
    llmopenai.WithFileSearch("vs_abc123"),
    llmopenai.WithCodeInterpreter(),
)
```

`WithWebSearchPreview` is also available for models that don't yet support
the newer `web_search` tool.

Groq — `browser_search`, `code_execution`, `visit_website` (requires a
`groq/compound*` model via the dedicated `NewCompoundLLM`):

```go
import llmgroq "github.com/joakimcarlsson/ai/llm/groq"

client := llmgroq.NewCompoundLLM(
    llmgroq.WithCompoundAPIKey(os.Getenv("GROQ_API_KEY")),
    llmgroq.WithCompoundModel(model.Model{APIModel: "groq/compound"}),
    llmgroq.WithBrowserSearch(llmgroq.BrowserSearchOpts{
        Country:       "us",
        IncludeImages: true,
    }),
    llmgroq.WithCodeExecution(),
    llmgroq.WithVisitWebsite(),
)
```

The regular `llmgroq.NewLLM` wrapper stays available for OpenAI-compatible
chat without built-ins.

## Cross-vendor wrappers

`llm/azure` (Azure OpenAI), `llm/vertexai` (Gemini on Vertex), and
`llm/bedrock` (Claude on Bedrock) are thin wrappers that delegate to their
underlying vendor module:

```go
import llmazure "github.com/joakimcarlsson/ai/llm/azure"

client := llmazure.NewLLM(
    llmazure.WithAPIKey(os.Getenv("AZURE_OPENAI_KEY")),
    llmazure.WithEndpoint("https://my-resource.openai.azure.com"),
    llmazure.WithAPIVersion("2024-02-01"),
    llmazure.WithModel(model.OpenAIModels[model.GPT4o]),
)
```

```go
import llmbedrock "github.com/joakimcarlsson/ai/llm/bedrock"

// Region is read from $AWS_REGION (or $AWS_DEFAULT_REGION).
client := llmbedrock.NewLLM(
    llmbedrock.WithModel(model.AnthropicModels[model.Claude45Sonnet]),
    llmbedrock.WithMaxTokens(2000),
)
```

```go
import llmvertex "github.com/joakimcarlsson/ai/llm/vertexai"

client := llmvertex.NewLLM(
    llmvertex.WithProject(os.Getenv("VERTEXAI_PROJECT")),
    llmvertex.WithLocation(os.Getenv("VERTEXAI_LOCATION")),
    llmvertex.WithModel(model.GeminiModels[model.Gemini25Pro]),
)
```

## OpenAI-compatible providers (BYOM)

OpenRouter, xAI, Mistral, Ollama, LocalAI, etc. — point `llm/openai` at the
right base URL:

```go
xai := llmopenai.NewLLM(
    llmopenai.WithAPIKey(os.Getenv("XAI_API_KEY")),
    llmopenai.WithBaseURL("https://api.x.ai/v1"),
    llmopenai.WithModel(model.XAIModels[model.Grok2]),
)
```

Groq is published as its own module (`llm/groq`) so it can expose compound-model
built-ins on top of the OpenAI-compatible surface. Use `llmgroq.NewLLM` for the
thin wrapper or `llmgroq.NewCompoundLLM` for built-in tools.

For a managed registry of these, see [BYOM](../advanced/byom.md).

## Tracing

Every vendor's `NewLLM(...)` returns a tracing-wrapped client. Spans + metrics
are emitted automatically via OpenTelemetry. See [Tracing](../advanced/tracing.md)
for setup.
