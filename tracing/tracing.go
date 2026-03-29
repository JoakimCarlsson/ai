// Package tracing provides OpenTelemetry instrumentation for AI provider calls and agent execution.
package tracing

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/joakimcarlsson/ai"

// Attr is an alias for attribute.KeyValue.
type Attr = attribute.KeyValue

// Span is an alias for trace.Span.
type Span = trace.Span

var (
	tracer     trace.Tracer
	tracerOnce sync.Once
)

// Tracer returns the shared OpenTelemetry tracer instance.
func Tracer() trace.Tracer {
	tracerOnce.Do(func() {
		tracer = otel.Tracer(instrumentationName)
	})
	return tracer
}

// GenAI semantic convention attribute keys.
const (
	AttrOperationName    = attribute.Key("gen_ai.operation.name")
	AttrSystem           = attribute.Key("gen_ai.system")
	AttrRequestModel     = attribute.Key("gen_ai.request.model")
	AttrRequestMaxTokens = attribute.Key(
		"gen_ai.request.max_tokens",
	)
	AttrRequestTemperature = attribute.Key(
		"gen_ai.request.temperature",
	)
	AttrRequestTopP          = attribute.Key("gen_ai.request.top_p")
	AttrResponseFinishReason = attribute.Key(
		"gen_ai.response.finish_reason",
	)
	AttrUsageInputTokens   = attribute.Key("gen_ai.usage.input_tokens")
	AttrUsageOutputTokens  = attribute.Key("gen_ai.usage.output_tokens")
	AttrUsageCacheCreation = attribute.Key(
		"gen_ai.usage.cache_creation_tokens",
	)
	AttrUsageCacheRead      = attribute.Key("gen_ai.usage.cache_read_tokens")
	AttrToolName            = attribute.Key("gen_ai.tool.name")
	AttrToolCallID          = attribute.Key("gen_ai.tool.call_id")
	AttrAgentName           = attribute.Key("gen_ai.agent.name")
	AttrAgentTotalTurns     = attribute.Key("gen_ai.agent.total_turns")
	AttrAgentTotalToolCalls = attribute.Key("gen_ai.agent.total_tool_calls")
	AttrToolCount           = attribute.Key("gen_ai.request.tool_count")
	AttrToolCallCount       = attribute.Key("gen_ai.response.tool_call_count")
	AttrInputCount          = attribute.Key("gen_ai.request.input_count")
	AttrDocumentCount       = attribute.Key("gen_ai.request.document_count")
	AttrResultCount         = attribute.Key("gen_ai.response.result_count")
	AttrUsageTotalTokens    = attribute.Key("gen_ai.usage.total_tokens")
	AttrUsageCharacters     = attribute.Key("gen_ai.usage.characters")
	AttrDurationSec         = attribute.Key("gen_ai.response.duration_sec")
	AttrLanguage            = attribute.Key("gen_ai.response.language")
)

// StartSpan creates a new client span with the given name and attributes.
func StartSpan(
	ctx context.Context,
	name string,
	attrs ...Attr,
) (context.Context, Span) {
	return Tracer().Start(ctx, name,
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// StartGenerateSpan creates a span for an LLM generate_content call.
func StartGenerateSpan(
	ctx context.Context,
	modelName string,
	system string,
	extra ...Attr,
) (context.Context, Span) {
	attrs := []Attr{
		AttrOperationName.String("generate_content"),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	}
	attrs = append(attrs, extra...)
	return StartSpan(
		ctx,
		fmt.Sprintf("generate_content %s", modelName),
		attrs...,
	)
}

// StartEmbeddingSpan creates a span for an embedding generation call.
func StartEmbeddingSpan(
	ctx context.Context,
	modelName string,
	system string,
) (context.Context, Span) {
	return StartSpan(ctx,
		fmt.Sprintf("generate_embeddings %s", modelName),
		AttrOperationName.String("generate_embeddings"),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	)
}

// StartRerankSpan creates a span for a reranking call.
func StartRerankSpan(
	ctx context.Context,
	modelName string,
	system string,
) (context.Context, Span) {
	return StartSpan(ctx,
		fmt.Sprintf("rerank %s", modelName),
		AttrOperationName.String("rerank"),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	)
}

// StartAudioSpan creates a span for an audio generation call.
func StartAudioSpan(
	ctx context.Context,
	modelName string,
	system string,
) (context.Context, Span) {
	return StartSpan(ctx,
		fmt.Sprintf("generate_audio %s", modelName),
		AttrOperationName.String("generate_audio"),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	)
}

// StartImageSpan creates a span for an image generation call.
func StartImageSpan(
	ctx context.Context,
	modelName string,
	system string,
) (context.Context, Span) {
	return StartSpan(ctx,
		fmt.Sprintf("generate_image %s", modelName),
		AttrOperationName.String("generate_image"),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	)
}

// StartTranscribeSpan creates a span for a transcription or translation call.
func StartTranscribeSpan(
	ctx context.Context,
	modelName string,
	system string,
	operation string,
) (context.Context, Span) {
	return StartSpan(ctx,
		fmt.Sprintf("%s %s", operation, modelName),
		AttrOperationName.String(operation),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	)
}

// StartFIMSpan creates a span for a fill-in-the-middle completion call.
func StartFIMSpan(
	ctx context.Context,
	modelName string,
	system string,
	extra ...Attr,
) (context.Context, Span) {
	attrs := []Attr{
		AttrOperationName.String("fim_complete"),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	}
	attrs = append(attrs, extra...)
	return StartSpan(
		ctx,
		fmt.Sprintf("fim_complete %s", modelName),
		attrs...,
	)
}

// StartAgentSpan creates a span for an agent invocation.
func StartAgentSpan(
	ctx context.Context,
	agentName string,
) (context.Context, Span) {
	return StartSpan(ctx,
		fmt.Sprintf("invoke_agent %s", agentName),
		AttrOperationName.String("invoke_agent"),
		AttrAgentName.String(agentName),
	)
}

// StartToolSpan creates a span for a tool execution.
func StartToolSpan(
	ctx context.Context,
	toolName string,
	callID string,
) (context.Context, Span) {
	return StartSpan(ctx,
		fmt.Sprintf("execute_tool %s", toolName),
		AttrOperationName.String("execute_tool"),
		AttrToolName.String(toolName),
		AttrToolCallID.String(callID),
	)
}

// SetResponseAttrs sets completion-time attributes on a span.
func SetResponseAttrs(span Span, attrs ...Attr) {
	span.SetAttributes(attrs...)
}

// SetError records an error and sets error status on a span.
func SetError(span Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
