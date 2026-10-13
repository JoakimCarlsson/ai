# LLM Providers

Each native LLM vendor is published as its own Go module under `llm/`. The
client returned from a vendor's `NewLLM(...)` satisfies the `llm.LLM`
interface, so once you've constructed it the rest of your code is
vendor-agnostic.

## Creating a client

```go
import (
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

client := llmopenai.NewLLM(
    llmopenai.WithAPIKey("your-api-key"),
    llmopenai.WithModel(llmopenai.Models[llmopenai.GPT4o]),
    llmopenai.WithMaxTokens(1000),
)
```

For Anthropic instead:

```go
import llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"

client := llmanthropic.NewLLM(
    llmanthropic.WithAPIKey("..."),
    llmanthropic.WithModel(anthropic.Models[anthropic.Claude45Sonnet]),
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
msg.AddBinary("image/png", imageData)

response, err := client.SendMessages(ctx, []message.Message{msg}, nil)
```

## Common options

Every vendor exports the standard set:

```go
llmopenai.WithAPIKey("...")
llmopenai.WithModel(openai.Models[openai.GPT4o])
llmopenai.WithMaxTokens(2000)
llmopenai.WithTemperature(0.7)
llmopenai.WithTopP(0.9)
llmopenai.WithTopK(40)
llmopenai.WithStopSequences("STOP", "END")
llmopenai.WithTimeout(30 * time.Second)
llmopenai.WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired})
```

!!! note "`WithToolChoice`"
    `WithToolChoice` is shared by the OpenAI, Anthropic, and Gemini modules
    (OpenAI-compatible providers inherit it through `llm/openai`). It takes the
    vendor-neutral `llm.ToolChoice` type: `Mode` is `ToolChoiceAuto` (default),
    `ToolChoiceNone`, `ToolChoiceRequired`, or `ToolChoiceSpecific` with a `Name`.
    It maps to each provider's native field (`tool_choice` for OpenAI/Anthropic,
    `toolConfig.functionCallingConfig` for Gemini) and is emitted only when tools
    are supplied. `ToolChoiceSpecific` with an empty `Name` is rejected before the
    request is sent.

!!! note "`WithTopK` on the OpenAI client"
    OpenAI's and Azure's own APIs reject `top_k` (HTTP 400), so `llmopenai.WithTopK`
    is sent only when a custom base URL points at an OpenAI-compatible provider that
    accepts it (Together, OpenRouter, Fireworks, ...); against OpenAI or Azure proper
    it has no effect. Native providers (Anthropic, Gemini, Bedrock) honor `WithTopK`
    directly. `WithStopSequences` sends every sequence provided (the OpenAI client
    caps at the API's limit of 4).

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
llmopenai.WithLogitBias(map[string]int{"1212": 5, "50256": -100})  // bias/ban tokens by id
llmopenai.WithLogprobs(3)                                          // logprobs:true + top_logprobs:3
llmopenai.WithN(3)                                                 // n completions per request
```

!!! note "Sampling knobs that change the response shape"
    `WithLogitBias`, `WithLogprobs`, and `WithN` live on the OpenAI client and so
    also cover every OpenAI-compatible provider (Groq, OpenRouter, xAI, Together,
    Fireworks, DeepSeek, Mistral, Ollama). They are emitted only when set, and
    are **OpenAI-only**: Anthropic supports none of them, and Gemini's
    `candidateCount` (the `n` equivalent) is out of scope — those providers never
    receive the fields.

    - `WithLogitBias` maps token IDs (tokenizer ids, OpenAI's wire shape) to a
      bias from -100 (ban) to 100 (force).
    - `WithLogprobs(n)` requests per-token log probabilities with up to `n`
      alternatives per position; the result lands on `Response.LogProbs`
      (`[]llm.TokenLogProb`), nil when not requested.
    - `WithN(n)` requests `n` completions; all land on `Response.Choices`
      (`[]llm.Choice`). The top-level `Content` / `FinishReason` / `ToolCalls` /
      `LogProbs` mirror choice 0, so single-completion callers are unaffected
      (`Choices` is empty when `n` is unset or `1`). Streaming with `n > 1` is not
      supported — use the non-streaming `SendMessages` path.

    `logit_bias` is rejected by reasoning-tier models (the gpt-5 family) with an
    HTTP 400; use a classic chat model such as `gpt-4o-mini` when you need it.

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

Gemini context caching stores a system instruction, tools and history once
and bills later requests that reference it at the cached rate. Create the
cache through `llm.ContextCacheProvider`, which the clients from
`llmgemini.NewLLM` and `llmvertexai.NewLLM` implement, then attach it to a
client. Requests that use a cache must not pass system messages or tools of
their own; they fail with `llmgemini.ErrCachedContentConflict`.

```go
creator := llmgemini.NewLLM(
    llmgemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    llmgemini.WithModel(model),
    llmgemini.WithCacheTTL(time.Hour),
    llmgemini.WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceAuto}),
)
name, err := creator.(llm.ContextCacheProvider).CreateCache(
    ctx, []message.Message{message.NewSystemMessage(longPrompt)}, tools,
)

client := llmgemini.NewLLM(
    llmgemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    llmgemini.WithModel(model),
    llmgemini.WithCachedContent(name),
)
resp, err := client.SendMessages(ctx, userMessages, nil)
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
    llmopenai.WithResponsesModel(openai.Models[openai.GPT5]),
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
    llmgroq.WithCompoundModel(llm.Model{APIModel: "groq/compound"}),
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

xAI — `web_search`, `x_search`, `code_execution` via the Responses API (use
`NewResponsesLLM` instead of `NewLLM`):

```go
import llmxai "github.com/joakimcarlsson/ai/llm/xai"

client := llmxai.NewResponsesLLM(
    llmxai.WithResponsesAPIKey(os.Getenv("XAI_API_KEY")),
    llmxai.WithResponsesModel(xai.Models[xai.Grok4]),
    llmxai.WithWebSearch(llmxai.WebSearchOpts{
        SearchContextSize: llmxai.SearchContextMedium,
    }),
    llmxai.WithXSearch(llmxai.XSearchOpts{
        AllowedXHandles: []string{"xai"},
        FromDate:        "2026-01-01",
    }),
    llmxai.WithCodeExecution(),
)
```

The thin `llmxai.NewLLM` wrapper remains available for OpenAI-compatible
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
    llmazure.WithDeployment("my-chat-deployment"),
)
```

```go
import llmbedrock "github.com/joakimcarlsson/ai/llm/bedrock"

// Region is read from $AWS_REGION (or $AWS_DEFAULT_REGION).
client := llmbedrock.NewLLM(
    llmbedrock.WithModel(anthropic.Models[anthropic.Claude45Sonnet]),
    llmbedrock.WithMaxTokens(2000),
)
```

Prompt caching is on by default on Bedrock: the underlying Anthropic client's
`cache_control` breakpoints reach Bedrock and populate `CacheReadTokens` /
`CacheCreationTokens` in the response usage. Pass `llmbedrock.WithDisableCache()`
to opt out. (Newer Claude models require at least 4096 cached tokens per
checkpoint before a cache hit is recorded.)

```go
import llmvertex "github.com/joakimcarlsson/ai/llm/vertexai"

client := llmvertex.NewLLM(
    llmvertex.WithProject(os.Getenv("VERTEXAI_PROJECT")),
    llmvertex.WithLocation(os.Getenv("VERTEXAI_LOCATION")),
    llmvertex.WithModel(llmvertex.Models[llmvertex.Gemini31Pro]),
)
```

## OpenAI-compatible providers (BYOM)

OpenRouter, Mistral, Ollama, LocalAI, etc. — point `llm/openai` at the right
base URL:

```go
openrouter := llmopenai.NewLLM(
    llmopenai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    llmopenai.WithBaseURL("https://openrouter.ai/api/v1"),
    llmopenai.WithModel(openai.Models[openai.GPT5]),
)
```

Groq and xAI are published as their own modules (`llm/groq`, `llm/xai`) so
they can expose vendor-specific built-in tools on top of the OpenAI-compatible
surface. Use the thin `NewLLM` constructor in each for plain chat, or the
dedicated `NewCompoundLLM` / `NewResponsesLLM` for built-in tools.

Berget AI (Swedish, EU-hosted; open-weight models) ships as `llm/berget`, a
thin wrapper pinned to `https://api.berget.ai/v1`. Pricing in the `model`
catalog is in EUR (`BergetModels`, `BergetEmbeddingModels`,
`BergetRerankerModels`, `BergetTranscriptionModels`):

Like the other OpenAI-compatible wrappers, `llm/berget` aliases `Option` but
does not re-export the option constructors; pass the standard ones from
`llm/openai`:

```go
import (
    llmberget "github.com/joakimcarlsson/ai/llm/berget"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

client := llmberget.NewLLM(
    llmopenai.WithAPIKey(os.Getenv("BERGET_API_KEY")),
    llmopenai.WithModel(llmberget.Models[llmberget.GPTOSS120B]),
    llmopenai.WithMaxTokens(1000),
)
```

For a managed registry of these, see [BYOM](../advanced/byom.md).

## Tracing

Every vendor's `NewLLM(...)` returns a tracing-wrapped client. Spans + metrics
are emitted automatically via OpenTelemetry. See [Tracing](../advanced/tracing.md)
for setup.

## Reasoning effort and usage

Leaving effort and thinking options unset sends no reasoning parameter; the
provider default applies. Options are sent only for models with `CanReason`.
Accepted values depend on the model, including whether thinking can be disabled.
OpenAI-compatible wrappers accept `openai.Option`, so their effort control is
`openai.WithReasoningEffort` unless a provider option is listed below.

`Usage.ReasoningTokens` is part of `Usage.OutputTokens`, including during
streaming. Add output tokens to a total only once. Gemini and Vertex AI return
candidate and thought counts separately; the client combines them into output
usage. Perplexity Sonar Deep Research also reports a separate reasoning count,
which is included once in output usage, but charges it at a different rate.
Its exact bill requires the provider's cost breakdown rather than pricing all
output tokens at the model's one output rate.

| Provider | Control and lowest/off setting | Reasoning-token count |
| --- | --- | --- |
| [OpenAI](https://github.com/openai/openai-go/blob/main/responses/response.go) | `WithReasoningEffort` / `WithResponsesReasoningEffort`; `ReasoningEffortNone` disables reasoning where supported | Chat `completion_tokens_details.reasoning_tokens`; Responses `output_tokens_details.reasoning_tokens` |
| [Azure](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/reasoning) | `WithReasoningEffort(openai.ReasoningEffortNone)`; inferred deployment capabilities or `WithReasoning(true)` gate it | Inherits OpenAI Chat usage |
| [xAI](https://docs.x.ai/developers/model-capabilities/text/reasoning) | Chat uses OpenAI effort; Responses uses `WithResponsesReasoningEffort(xai.ReasoningEffortLow)`. Supports medium/high/xhigh where accepted; reasoning cannot be disabled | Inherits Chat details; Responses `output_tokens_details.reasoning_tokens` |
| [Anthropic](https://github.com/anthropics/anthropic-sdk-go/blob/main/message.go) | `WithReasoningEffort(ReasoningEffortLow)` minimizes adaptive thinking; `WithThinkingBudget(0)` explicitly disables it. Positive budgets enable manual thinking on supported models and override adaptive effort | `output_tokens_details.thinking_tokens`, including the final streaming message delta |
| [Bedrock](https://docs.aws.amazon.com/bedrock/latest/userguide/claude-messages-extended-thinking.html) | Claude forwards `WithReasoningEffort` and `WithThinkingBudget` to Anthropic; budget zero disables thinking | Inherits Claude's thinking count when Bedrock supplies it. Other catalog vendors are not implemented by this client |
| [Gemini](https://ai.google.dev/gemini-api/docs/thinking) | `WithThinkingLevel(ThinkingLevelMinimal)` is the lowest level; `WithThinkingBudget(0)` disables thinking on supported models (not 2.5 Pro). Budget `-1` is dynamic | `thoughtsTokenCount`; output combines candidates and thoughts |
| [Vertex AI](https://cloud.google.com/vertex-ai/generative-ai/docs/thinking) | Forwards `WithThinkingLevel` and `WithThinkingBudget` to Gemini | Inherits Gemini usage |
| [DeepSeek](https://api-docs.deepseek.com/guides/thinking_mode/) | `WithReasoningEffort(ReasoningEffortNone)` / `WithThinking(false)` disables thinking; low/high/max set depth on models that accept them | Inherits Chat details; top-level `reasoning_tokens` is also recognized when details are absent |
| [Groq](https://console.groq.com/docs/api-reference) | OpenAI effort; none disables supported Qwen models, low is the lowest GPT-OSS level | `completion_tokens_details.reasoning_tokens` through the OpenAI client |
| [Cerebras](https://inference-docs.cerebras.ai/capabilities/reasoning) | OpenAI effort; none disables supported models, low minimizes GPT-OSS reasoning; always-thinking models can ignore the control | `completion_tokens_details.reasoning_tokens` through the OpenAI client |
| [OpenRouter](https://openrouter.ai/docs/api/api-reference/chat/send-chat-completion-request) | OpenAI effort shorthand maps to `reasoning.effort`; none requests no reasoning where the routed model supports it | `completion_tokens_details.reasoning_tokens` through the OpenAI client |
| [Perplexity](https://github.com/perplexityai/perplexity-py/blob/main/src/perplexity/types/shared/usage_info.py) | OpenAI effort; minimal/low minimize reasoning where supported; no documented off value for Sonar Deep Research | Top-level `reasoning_tokens`; [Sonar Deep Research bills it separately](https://docs.perplexity.ai/docs/getting-started/pricing) |
| [Together](https://docs.together.ai/reference/chat-completions) | OpenAI effort low/medium/high; `WithThinking(false)` sends `reasoning.enabled=false` on models supporting the toggle | No separate reasoning count in the documented completion usage |
| [Fireworks](https://docs.fireworks.ai/api-reference/post-chatcompletions) | OpenAI effort; none disables reasoning on supported models, low minimizes always-thinking models | No separate reasoning count in the documented completion usage |
| [Mistral](https://docs.mistral.ai/studio/conversations/reasoning) | OpenAI effort; none minimizes thinking and omits the trace on supported Mistral models; hosted models may require low instead | No separate reasoning count in the documented completion usage |
| [Ollama](https://github.com/ollama/ollama/blob/main/docs/api/openai-compatibility.mdx) | OpenAI effort; none maps to disabled thinking where supported, low is the lowest GPT-OSS level | The compatible endpoint reports completion totals without a separate thinking count |
| [Berget](https://api.berget.ai/) | The documented compatible API has no effort/thinking control; model defaults apply | No documented separate reasoning count |

Providers without a documented breakdown leave `ReasoningTokens` at zero.
The compatible client still maps standard reasoning details if an endpoint
supplies them; it does not estimate counts from reasoning text.

Azure Responses effort forwarding depends on the Azure Responses constructor
in [#294](https://github.com/JoakimCarlsson/ai/issues/294), which is not present
in this checkout. OpenAI Responses usage and effort handling are ready for
that constructor to reuse.
