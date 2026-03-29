# OpenTelemetry Tracing

Built-in OpenTelemetry instrumentation for all provider calls and agent execution. Includes traces, metrics, and structured log records following [GenAI semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/). When no providers are configured, everything is a zero-cost no-op.

## Setup

Configure a global `TracerProvider` before making any calls. The library automatically creates spans for every operation.

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
defer tp.Shutdown(ctx)

otel.SetTracerProvider(tp)
```

That's it. All subsequent LLM calls, tool executions, and agent runs will produce spans and metrics.

## Span Hierarchy

When using the agent framework, spans form a parent-child tree:

```
invoke_agent {agent_name}
├── generate_content {model}       (LLM turn 1)
├── execute_tool {tool_name}       (tool call)
├── generate_content {model}       (LLM turn 2)
└── ...
```

When using providers standalone (no agent), each call produces a root span:

```
generate_content {model}
generate_embeddings {model}
rerank {model}
generate_audio {model}
generate_image {model}
transcribe {model}
fim_complete {model}
```

## Instrumented Operations

Every provider package is instrumented at the public API level — one span per call, covering all underlying providers.

| Package | Span Name | Methods |
|---------|-----------|---------|
| `providers` (LLM) | `generate_content` | `SendMessages`, `StreamResponse`, and structured output variants |
| `embeddings` | `generate_embeddings` | `GenerateEmbeddings`, `GenerateMultimodalEmbeddings`, `GenerateContextualizedEmbeddings` |
| `rerankers` | `rerank` | `Rerank` |
| `audio` | `generate_audio` | `GenerateAudio`, `StreamAudio` |
| `image_generation` | `generate_image` | `GenerateImage`, `GenerateImageStreaming` |
| `transcription` | `transcribe` / `translate` | `Transcribe`, `Translate` |
| `fim` | `fim_complete` | `Complete`, `CompleteStream` |
| `agent` | `invoke_agent` | `Chat`, `ChatStream`, `Continue`, `ContinueStream` |
| `agent` (tools) | `execute_tool` | Each tool call during agent execution |

## Span Attributes

Spans carry GenAI semantic convention attributes.

### LLM (`generate_content`)

| Attribute | When |
|-----------|------|
| `gen_ai.system` | Always |
| `gen_ai.request.model` | Always |
| `gen_ai.request.max_tokens` | Always |
| `gen_ai.request.temperature` | If set |
| `gen_ai.request.top_p` | If set |
| `gen_ai.usage.input_tokens` | On completion |
| `gen_ai.usage.output_tokens` | On completion |
| `gen_ai.usage.cache_creation_tokens` | If non-zero |
| `gen_ai.usage.cache_read_tokens` | If non-zero |
| `gen_ai.response.finish_reason` | On completion |
| `gen_ai.response.tool_call_count` | If tool calls present |

### Agent (`invoke_agent`)

| Attribute | When |
|-----------|------|
| `gen_ai.agent.name` | Always |
| `gen_ai.usage.input_tokens` | On completion (aggregated) |
| `gen_ai.usage.output_tokens` | On completion (aggregated) |
| `gen_ai.agent.total_turns` | On completion |
| `gen_ai.agent.total_tool_calls` | On completion |

### Tool (`execute_tool`)

| Attribute | When |
|-----------|------|
| `gen_ai.tool.name` | Always |
| `gen_ai.tool.call_id` | Always |

## Streaming

Streaming calls (`StreamResponse`, `StreamAudio`, `CompleteStream`) are fully traced. The span covers the entire stream lifetime — from the initial call until the channel closes. Response attributes (token usage, finish reason) are recorded when the final event arrives.

## Metrics

Every provider call records two metrics via the global `MeterProvider`:

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `gen_ai.client.operation.duration` | Float64Histogram | `s` | Duration of each provider call |
| `gen_ai.client.token.usage` | Int64Counter | `{token}` | Token consumption per call |

Both metrics carry these attributes:

| Attribute | Description |
|-----------|-------------|
| `gen_ai.operation.name` | Operation type (`generate_content`, `generate_embeddings`, `rerank`, etc.) |
| `gen_ai.system` | Provider name (`openai`, `anthropic`, `voyage`, etc.) |
| `gen_ai.request.model` | Model identifier |
| `error.type` | Error message (only on failed calls) |

The token usage counter additionally carries `gen_ai.token.type` (`input` or `output`) to distinguish token direction. Token metrics are only recorded when the count is non-zero.

### Metrics Setup

Metrics work the same as traces — configure a global `MeterProvider`:

```go
import (
    "go.opentelemetry.io/otel"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)

exporter, _ := otlpmetrichttp.New(ctx)
mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(
    sdkmetric.NewPeriodicReader(exporter),
))
defer mp.Shutdown(ctx)

otel.SetMeterProvider(mp)
```

## Log Records

LLM calls emit OpenTelemetry log records tied to the active span. Log bodies are structured JSON following GenAI semantic conventions:

| Event Name | Body Structure |
|------------|----------------|
| `gen_ai.system.message` | `{"content": "..."}` |
| `gen_ai.user.message` | `{"content": "..."}` |
| `gen_ai.choice` | `{"index": 0, "content": "...", "finish_reason": "..."}` |

Log records require a global `LoggerProvider` to be configured. Without one, they are silently dropped.

### Content Capture

Message content is **elided by default** for privacy. To include actual message content in log records, set:

```bash
export OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT=true
```

When disabled, log bodies contain `<elided>` instead of the actual content.

## Retry Visibility

When a provider call is retried (rate limits, transient errors), each retry attempt is recorded as a span event on the `generate_content` span:

```
Event: "retry"
  attempt = 1
  retry_after_ms = 2000
  error = "429 Too Many Requests"
```

This gives visibility into retries without creating additional spans, making it easy to diagnose latency spikes caused by rate limiting.

## OTLP Export

To export traces to Jaeger, Grafana Tempo, Datadog, or any OTLP-compatible backend:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.36.0"
)

exporter, _ := otlptracehttp.New(ctx)

res, _ := resource.New(ctx,
    resource.WithAttributes(
        semconv.ServiceNameKey.String("my-ai-service"),
        semconv.ServiceVersionKey.String("1.0.0"),
    ),
)

tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithResource(res),
)
defer tp.Shutdown(ctx)

otel.SetTracerProvider(tp)
```

Configure the OTLP endpoint via environment variable:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

## Standalone Provider Tracing

Tracing works without the agent framework. Any provider call creates spans and records metrics automatically:

```go
otel.SetTracerProvider(tp)

client, _ := llm.NewLLM(model.ProviderAnthropic,
    llm.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    llm.WithModel(model.AnthropicModels[model.Claude4Sonnet]),
)

// This call produces a "generate_content claude-sonnet-4-6-20250514" span
// and records duration + token usage metrics
response, _ := client.SendMessages(ctx, messages, nil)
```

The same applies to embeddings, audio, image generation, transcription, rerankers, and FIM.

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"

    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
    "github.com/joakimcarlsson/ai/tool/functiontool"
)

func main() {
    ctx := context.Background()

    exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
    tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
    defer func() { _ = tp.Shutdown(ctx) }()
    otel.SetTracerProvider(tp)

    client, _ := llm.NewLLM(model.ProviderOpenAI,
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
    )

    timeTool := functiontool.New(
        "get_time",
        "Get the current time",
        func(_ context.Context, p struct{}) (string, error) {
            return "14:30 UTC", nil
        },
    )

    myAgent := agent.New(client,
        agent.WithTools(timeTool),
    )

    resp, err := myAgent.Chat(ctx, "What time is it?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Content)
}
```

This produces spans:

```
invoke_agent
├── generate_content gpt-5-nano
├── execute_tool get_time
└── generate_content gpt-5-nano
```
