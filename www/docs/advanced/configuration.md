# Configuration

## LLM Client Options

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

## Embedding Client Options

```go
embedder, err := embeddings.NewEmbedding(
    model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
    embeddings.WithBatchSize(100),
    embeddings.WithTimeout(30*time.Second),
    embeddings.WithVoyageOptions(
        embeddings.WithInputType("document"),
        embeddings.WithOutputDimension(1024),
        embeddings.WithOutputDtype("float"),
    ),
)
```

## Reranker Client Options

```go
reranker, err := rerankers.NewReranker(
    model.ProviderVoyage,
    rerankers.WithAPIKey(""),
    rerankers.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
    rerankers.WithTopK(10),
    rerankers.WithReturnDocuments(true),
    rerankers.WithTruncation(true),
    rerankers.WithTimeout(30*time.Second),
)
```

## Image Generation Client Options

```go
// OpenAI/xAI
client, err := image_generation.NewImageGeneration(
    model.ProviderOpenAI,
    image_generation.WithAPIKey("your-key"),
    image_generation.WithModel(model.OpenAIImageGenerationModels[model.DALLE3]),
    image_generation.WithTimeout(60*time.Second),
    image_generation.WithOpenAIOptions(
        image_generation.WithOpenAIBaseURL("custom-endpoint"),
    ),
)

// Gemini
client, err := image_generation.NewImageGeneration(
    model.ProviderGemini,
    image_generation.WithAPIKey("your-key"),
    image_generation.WithModel(model.GeminiImageGenerationModels[model.Imagen4]),
    image_generation.WithTimeout(60*time.Second),
    image_generation.WithGeminiOptions(
        image_generation.WithGeminiBackend(genai.BackendVertexAI),
    ),
)
```

## Audio Generation Client Options

```go
client, err := audio.NewAudioGeneration(
    model.ProviderElevenLabs,
    audio.WithAPIKey("your-key"),
    audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
    audio.WithTimeout(30*time.Second),
    audio.WithElevenLabsOptions(
        audio.WithElevenLabsBaseURL("custom-endpoint"),
    ),
)
```

## Speech-to-Text Client Options

```go
client, err := transcription.NewSpeechToText(
    model.ProviderOpenAI,
    transcription.WithAPIKey("your-key"),
    transcription.WithModel(model.OpenAITranscriptionModels[model.GPT4oTranscribe]),
    transcription.WithTimeout(30*time.Second),
)
```

## Provider-Specific Options

```go
// Anthropic
llm.WithAnthropicOptions(
    llm.WithAnthropicBeta("beta-feature"),
    llm.WithAnthropicBedrock(true),
    llm.WithAnthropicDisableCache(),
    llm.WithAnthropicReasoningEffort(llm.AnthropicReasoningEffortMedium),
)

// OpenAI
llm.WithOpenAIOptions(
    llm.WithOpenAIBaseURL("custom-endpoint"),
    llm.WithOpenAIExtraHeaders(map[string]string{"Custom-Header": "value"}),
    llm.WithOpenAIDisableCache(),
    llm.WithReasoningEffort(llm.OpenAIReasoningEffortHigh),
    llm.WithOpenAIFrequencyPenalty(0.5),
    llm.WithOpenAIPresencePenalty(0.3),
    llm.WithOpenAISeed(42),
    llm.WithOpenAIParallelToolCalls(false),
)

// Gemini
llm.WithGeminiOptions(
    llm.WithGeminiDisableCache(),
    llm.WithGeminiFrequencyPenalty(0.5),
    llm.WithGeminiPresencePenalty(0.3),
    llm.WithGeminiSeed(42),
)

// Azure OpenAI
llm.WithAzureOptions(
    llm.WithAzureEndpoint("https://your-resource.openai.azure.com"),
    llm.WithAzureAPIVersion("2024-02-15-preview"),
)

// Bedrock (via Anthropic)
llm.WithAnthropicOptions(
    llm.WithAnthropicBedrock(true),
)
llm.WithBedrockOptions(...)
```

## Retry Configuration

All LLM providers include automatic retry with exponential backoff and jitter. Each provider has optimized defaults:

```go
// Default retry config (used by most providers)
llm.DefaultRetryConfig()   // retries: 429, 500, 502, 503, 504

// Provider-specific configs
llm.OpenAIRetryConfig()     // retries: 429, 500
llm.AnthropicRetryConfig()  // retries: 429, 529
llm.GeminiRetryConfig()     // no Retry-After header support
llm.MistralRetryConfig()    // retries: 429, 500, 502, 503
```

| Setting | Default | Description |
|---------|---------|-------------|
| `MaxRetries` | 3 | Maximum retry attempts |
| `BaseBackoffMs` | 2000 | Initial backoff in milliseconds |
| `JitterPercent` | 0.2 | Jitter added to backoff (20%) |
| `RetryStatusCodes` | varies | HTTP status codes that trigger retries |
| `CheckRetryAfter` | true | Respect the `Retry-After` header |

Retries use exponential backoff: `base * 2^(attempt-1) + jitter`. When `CheckRetryAfter` is enabled and the server sends a `Retry-After` header, that value takes precedence.

## Agent Options

See the [Agent Framework Overview](../agent/overview.md) for a full table of agent configuration options.
