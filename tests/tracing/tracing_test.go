package tracing

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel/codes"

	"github.com/joakimcarlsson/ai/tracing"
)

func TestAllSpanStarters(t *testing.T) {
	exporter := setupTracing(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		fn     func()
		prefix string
		opName string
	}{
		{
			name: "embedding",
			fn: func() {
				_, s := tracing.StartEmbeddingSpan(
					ctx, "voyage-3", "voyage",
				)
				s.End()
			},
			prefix: "generate_embeddings",
			opName: "generate_embeddings",
		},
		{
			name: "rerank",
			fn: func() {
				_, s := tracing.StartRerankSpan(
					ctx, "rerank-2", "voyage",
				)
				s.End()
			},
			prefix: "rerank",
			opName: "rerank",
		},
		{
			name: "audio",
			fn: func() {
				_, s := tracing.StartAudioSpan(
					ctx, "eleven-turbo", "elevenlabs",
				)
				s.End()
			},
			prefix: "generate_audio",
			opName: "generate_audio",
		},
		{
			name: "image",
			fn: func() {
				_, s := tracing.StartImageSpan(
					ctx, "dall-e-3", "openai",
				)
				s.End()
			},
			prefix: "generate_image",
			opName: "generate_image",
		},
		{
			name: "transcribe",
			fn: func() {
				_, s := tracing.StartTranscribeSpan(
					ctx, "whisper-1", "openai", "transcribe",
				)
				s.End()
			},
			prefix: "transcribe",
			opName: "transcribe",
		},
		{
			name: "fim",
			fn: func() {
				_, s := tracing.StartFIMSpan(
					ctx, "codestral", "mistral",
				)
				s.End()
			},
			prefix: "fim_complete",
			opName: "fim_complete",
		},
	}

	for _, tc := range tests {
		exporter.Reset()
		tc.fn()
		spans := exporter.GetSpans()
		span := findSpan(spans, tc.prefix)
		if span == nil {
			t.Errorf(
				"%s: expected span with prefix %q",
				tc.name,
				tc.prefix,
			)
			continue
		}
		if spanAttr(span, "gen_ai.operation.name") != tc.opName {
			t.Errorf(
				"%s: expected operation.name %q, got %q",
				tc.name,
				tc.opName,
				spanAttr(span, "gen_ai.operation.name"),
			)
		}
		if spanAttr(span, "gen_ai.request.model") == "" {
			t.Errorf(
				"%s: expected request.model attribute",
				tc.name,
			)
		}
		if spanAttr(span, "gen_ai.system") == "" {
			t.Errorf("%s: expected system attribute", tc.name)
		}
	}
}

func TestSetError_NilError(t *testing.T) {
	exporter := setupTracing(t)
	_, span := tracing.StartSpan(
		context.Background(),
		"test",
	)
	tracing.SetError(span, nil)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected span")
	}
	if spans[0].Status.Code == codes.Error {
		t.Error("expected no error status when err is nil")
	}
}

func TestSetResponseAttrs(t *testing.T) {
	exporter := setupTracing(t)
	_, span := tracing.StartSpan(
		context.Background(),
		"test",
	)
	tracing.SetResponseAttrs(span,
		tracing.AttrUsageInputTokens.Int64(42),
		tracing.AttrResponseFinishReason.String("end_turn"),
	)
	span.End()

	spans := exporter.GetSpans()
	if spanAttrInt(&spans[0], "gen_ai.usage.input_tokens") != 42 {
		t.Error("expected input_tokens 42")
	}
	if spanAttr(
		&spans[0],
		"gen_ai.response.finish_reason",
	) != "end_turn" {
		t.Error("expected finish_reason end_turn")
	}
}

func TestStartGenerateSpan_WithExtraAttrs(t *testing.T) {
	exporter := setupTracing(t)
	temp := 0.7
	_, span := tracing.StartGenerateSpan(
		context.Background(),
		"gpt-4",
		"openai",
		tracing.AttrRequestMaxTokens.Int64(1000),
		tracing.AttrRequestTemperature.Float64(temp),
	)
	span.End()

	spans := exporter.GetSpans()
	s := findSpan(spans, "generate_content")
	if s == nil {
		t.Fatal("expected generate_content span")
	}
	if spanAttrInt(s, "gen_ai.request.max_tokens") != 1000 {
		t.Error("expected max_tokens 1000")
	}
}

func TestSetError_WithError(t *testing.T) {
	exporter := setupTracing(t)
	_, span := tracing.StartSpan(
		context.Background(),
		"test",
	)
	tracing.SetError(span, fmt.Errorf("something broke"))
	span.End()

	spans := exporter.GetSpans()
	if spans[0].Status.Code != codes.Error {
		t.Error("expected error status")
	}
	if spans[0].Status.Description != "something broke" {
		t.Errorf(
			"expected error description 'something broke', got %q",
			spans[0].Status.Description,
		)
	}
}
