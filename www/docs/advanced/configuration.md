# Configuration

After the modular split, every option lives on the vendor module. Each
`llm/<vendor>`, `embeddings/<vendor>`, etc. exports its own `Options` type
and `WithXxx` helpers — there's no generic factory or provider switch.

## LLM client options

Each LLM vendor module exports a `NewLLM` plus shared and vendor-specific
options.

### OpenAI

```go
import (
    "github.com/joakimcarlsson/ai/llm"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/model"
)

client := llmopenai.NewLLM(
    llmopenai.WithAPIKey("your-key"),
    llmopenai.WithModel(model.OpenAIModels[model.GPT4o]),
    llmopenai.WithMaxTokens(2000),
    llmopenai.WithTemperature(0.7),
    llmopenai.WithTopP(0.9),
    llmopenai.WithTimeout(30*time.Second),
    llmopenai.WithStopSequences("STOP", "END"),
    llmopenai.WithBaseURL("https://custom-endpoint"),
    llmopenai.WithExtraHeaders(map[string]string{"X-Header": "value"}),
    llmopenai.WithDisableCache(),
    llmopenai.WithReasoningEffort(llmopenai.ReasoningEffortHigh),
    llmopenai.WithFrequencyPenalty(0.5),
    llmopenai.WithPresencePenalty(0.3),
    llmopenai.WithSeed(42),
    llmopenai.WithParallelToolCalls(false),
    llmopenai.WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
    llmopenai.WithLogitBias(map[string]int{"50256": -100}),
    llmopenai.WithLogprobs(3),
    llmopenai.WithN(3),
)
```

`WithLogitBias`, `WithLogprobs`, and `WithN` are OpenAI-only sampling knobs
(inherited by every OpenAI-compatible provider) and are emitted only when set.
`WithLogitBias` biases or bans tokens by tokenizer id (-100 to 100).
`WithLogprobs(n)` surfaces per-token log probabilities with up to `n`
alternatives on `Response.LogProbs`. `WithN(n)` requests `n` completions on
`Response.Choices`, with the top-level fields mirroring choice 0 (streaming with
`n > 1` is unsupported). `logit_bias` is rejected by reasoning-tier models (the
gpt-5 family); use a classic chat model such as `gpt-4o-mini` for it.

`WithToolChoice` controls whether and which tool the model may call. It takes
the shared `llm.ToolChoice` type — `Mode` is one of `ToolChoiceAuto` (default),
`ToolChoiceNone`, `ToolChoiceRequired`, or `ToolChoiceSpecific` (which also sets
`Name`). The field is only emitted when tools are supplied, and is available on
the OpenAI, Anthropic, and Gemini modules (OpenAI-compatible providers inherit
it through `llm/openai`).

### Anthropic

```go
import llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"

client := llmanthropic.NewLLM(
    llmanthropic.WithAPIKey("your-key"),
    llmanthropic.WithModel(model.AnthropicModels[model.Claude45Sonnet]),
    llmanthropic.WithMaxTokens(4000),
    llmanthropic.WithTemperature(0.7),
    llmanthropic.WithBeta("beta-feature"),
    llmanthropic.WithBedrock(true),
    llmanthropic.WithDisableCache(),
    llmanthropic.WithReasoningEffort(llmanthropic.ReasoningEffortMedium),
    llmanthropic.WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
)
```

### Gemini

```go
import llmgemini "github.com/joakimcarlsson/ai/llm/gemini"

client := llmgemini.NewLLM(
    llmgemini.WithAPIKey("your-key"),
    llmgemini.WithModel(model.GeminiModels[model.Gemini25Flash]),
    llmgemini.WithMaxTokens(2000),
    llmgemini.WithDisableCache(),
    llmgemini.WithFrequencyPenalty(0.5),
    llmgemini.WithPresencePenalty(0.3),
    llmgemini.WithSeed(42),
    llmgemini.WithThinkingLevel(llmgemini.ThinkingLevelHigh),
    llmgemini.WithToolChoice(llm.ToolChoice{Mode: llm.ToolChoiceRequired}),
)
```

### Azure OpenAI

```go
import llmazure "github.com/joakimcarlsson/ai/llm/azure"

client := llmazure.NewLLM(
    llmazure.WithAPIKey("your-key"),
    llmazure.WithEndpoint("https://your-resource.openai.azure.com"),
    llmazure.WithDeployment("my-chat-deployment"),
)
```

### Bedrock

```go
import llmbedrock "github.com/joakimcarlsson/ai/llm/bedrock"

client := llmbedrock.NewLLM(
    llmbedrock.WithRegion("us-east-1"),
    llmbedrock.WithModel(model.BedrockModels[model.BedrockClaude45Sonnet]),
)
```

## Embedding client options

```go
import embvoyage "github.com/joakimcarlsson/ai/embeddings/voyage"

embedder := embvoyage.NewEmbedding(
    embvoyage.WithAPIKey(""),
    embvoyage.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
    embvoyage.WithBatchSize(100),
    embvoyage.WithTimeout(30*time.Second),
    embvoyage.WithInputType("document"),
    embvoyage.WithOutputDimension(1024),
    embvoyage.WithOutputDtype("float"),
)
```

The same shape applies to `embeddings/openai`, `embeddings/gemini`,
`embeddings/cohere`, `embeddings/mistral`, `embeddings/jina`. Vendor-specific
options live on the vendor's module.

## Reranker client options

```go
import rerankvoyage "github.com/joakimcarlsson/ai/rerankers/voyage"

reranker := rerankvoyage.NewReranker(
    rerankvoyage.WithAPIKey(""),
    rerankvoyage.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
    rerankvoyage.WithTopK(10),
    rerankvoyage.WithReturnDocuments(true),
    rerankvoyage.WithTruncation(true),
    rerankvoyage.WithTimeout(30*time.Second),
)
```

## Image generation client options

```go
import (
    imageopenai "github.com/joakimcarlsson/ai/image/openai"
    imagegemini "github.com/joakimcarlsson/ai/image/gemini"
    "google.golang.org/genai"
)

// OpenAI
client := imageopenai.NewGeneration(
    imageopenai.WithAPIKey("your-key"),
    imageopenai.WithModel(model.OpenAIImageGenerationModels[model.GPTImage15]),
    imageopenai.WithTimeout(60*time.Second),
    imageopenai.WithBaseURL("custom-endpoint"),
)

// Gemini / Vertex AI
client := imagegemini.NewImageGeneration(
    imagegemini.WithAPIKey("your-key"),
    imagegemini.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
    imagegemini.WithTimeout(60*time.Second),
    imagegemini.WithBackend(genai.BackendVertexAI),
)
```

## TTS client options

```go
import ttseleven "github.com/joakimcarlsson/ai/tts/elevenlabs"

client := ttseleven.NewAudioGeneration(
    ttseleven.WithAPIKey("your-key"),
    ttseleven.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
    ttseleven.WithTimeout(30*time.Second),
    ttseleven.WithBaseURL("custom-endpoint"),
)
```

## STT client options

```go
import sttopenai "github.com/joakimcarlsson/ai/stt/openai"

client := sttopenai.NewSpeechToText(
    sttopenai.WithAPIKey("your-key"),
    sttopenai.WithModel(model.OpenAITranscriptionModels[model.GPT4oTranscribe]),
    sttopenai.WithTimeout(30*time.Second),
)
```

## Retry configuration

Each LLM vendor module exposes its own retry config with provider-tuned
defaults. They all share the same field shape via `llm.RetryConfig`:

```go
import (
    "github.com/joakimcarlsson/ai/llm"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
)

// Defaults baked into each vendor
llmopenai.DefaultRetryConfig()      // retries: 429, 500
llmanthropic.DefaultRetryConfig()   // retries: 429, 529

// Override per-client
client := llmopenai.NewLLM(
    llmopenai.WithRetryConfig(llm.RetryConfig{
        MaxRetries:       5,
        BaseBackoffMs:    1000,
        JitterPercent:    0.2,
        RetryStatusCodes: []int{429, 500, 502, 503, 504},
        CheckRetryAfter:  true,
    }),
)
```

| Setting | Default | Description |
|---------|---------|-------------|
| `MaxRetries` | 3 | Maximum retry attempts |
| `BaseBackoffMs` | 2000 | Initial backoff in milliseconds |
| `JitterPercent` | 0.2 | Jitter added to backoff (20%) |
| `RetryStatusCodes` | varies | HTTP status codes that trigger retries |
| `CheckRetryAfter` | true | Respect the `Retry-After` header |

Retries use exponential backoff: `base * 2^(attempt-1) + jitter`. When
`CheckRetryAfter` is enabled and the server sends a `Retry-After` header,
that value takes precedence.

## Tracing wrappers

Every modality interface module exports a `WithTracing` helper that wraps
any vendor client without dragging an SDK into the import:

```go
import "github.com/joakimcarlsson/ai/llm"

traced := llm.WithTracing(client, llm.TracingAttrs{
    Provider: "openai",
    Model:    "gpt-4o",
})
```

See [Tracing](tracing.md) for the full picture.

## Agent options

See the [Agent Framework Overview](../agent/overview.md) for a full table of
agent configuration options.
