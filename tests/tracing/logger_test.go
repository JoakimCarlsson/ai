package tracing

import (
	"context"
	"os"
	"testing"

	"github.com/joakimcarlsson/ai/tracing"
)

func TestLogChoice_StructuredBody(t *testing.T) {
	exporter := setupTracing(t)
	capture := setupLogs(t)

	os.Setenv(
		"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
		"true",
	)
	t.Cleanup(func() {
		os.Unsetenv(
			"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
		)
	})

	ctx, span := tracing.StartGenerateSpan(
		context.Background(),
		"test-model",
		"test-system",
	)
	tracing.LogChoice(ctx, "hello world", "end_turn")
	span.End()

	_ = exporter

	records := capture.Records()
	var found bool
	for _, rec := range records {
		for _, attr := range rec.Attributes {
			if string(attr.Key) == "event.name" &&
				attr.Value.AsString() == "gen_ai.choice" {
				found = true
				kvs := rec.Body.AsMap()
				var hasFinishReason bool
				for _, kv := range kvs {
					if kv.Key == "finish_reason" {
						hasFinishReason = true
						if kv.Value.AsString() != "end_turn" {
							t.Errorf(
								"expected finish_reason 'end_turn', got %q",
								kv.Value.AsString(),
							)
						}
					}
				}
				if !hasFinishReason {
					t.Error("expected finish_reason in log body")
				}
			}
		}
	}
	if !found {
		t.Error("expected gen_ai.choice log record")
	}
}

func TestLogMessage_Elided(t *testing.T) {
	exporter := setupTracing(t)
	capture := setupLogs(t)

	os.Unsetenv(
		"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
	)

	ctx, span := tracing.StartGenerateSpan(
		context.Background(),
		"test-model",
		"test-system",
	)
	tracing.LogUserMessage(ctx, "secret data")
	span.End()

	_ = exporter

	records := capture.Records()
	for _, rec := range records {
		kvs := rec.Body.AsMap()
		for _, kv := range kvs {
			if kv.Key == "content" &&
				kv.Value.AsString() == "secret data" {
				t.Error(
					"expected content to be elided, got actual content",
				)
			}
		}
	}
}

func TestLogSystemMessage(t *testing.T) {
	exporter := setupTracing(t)
	capture := setupLogs(t)

	os.Setenv(
		"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
		"true",
	)
	t.Cleanup(func() {
		os.Unsetenv(
			"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
		)
	})

	ctx, span := tracing.StartGenerateSpan(
		context.Background(),
		"test-model",
		"test-system",
	)
	tracing.LogSystemMessage(ctx, "you are helpful")
	span.End()

	_ = exporter

	var found bool
	for _, rec := range capture.Records() {
		for _, attr := range rec.Attributes {
			if string(attr.Key) == "event.name" &&
				attr.Value.AsString() == "gen_ai.system.message" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected gen_ai.system.message log record")
	}
}

func TestEmitLog_NoSpanContext(t *testing.T) {
	setupTracing(t)
	capture := setupLogs(t)

	tracing.LogUserMessage(
		context.Background(),
		"no span here",
	)

	if len(capture.Records()) != 0 {
		t.Error(
			"expected no log records when no span context",
		)
	}
}

func TestLogUserMessage_WithContentCapture(t *testing.T) {
	exporter := setupTracing(t)
	capture := setupLogs(t)

	os.Setenv(
		"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
		"true",
	)
	t.Cleanup(func() {
		os.Unsetenv(
			"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
		)
	})

	ctx, span := tracing.StartGenerateSpan(
		context.Background(),
		"test-model",
		"test-system",
	)
	tracing.LogUserMessage(ctx, "hello there")
	span.End()

	_ = exporter

	var found bool
	for _, rec := range capture.Records() {
		for _, attr := range rec.Attributes {
			if string(attr.Key) == "event.name" &&
				attr.Value.AsString() == "gen_ai.user.message" {
				found = true
				kvs := rec.Body.AsMap()
				for _, kv := range kvs {
					if kv.Key == "content" &&
						kv.Value.AsString() != "hello there" {
						t.Errorf(
							"expected content 'hello there', got %q",
							kv.Value.AsString(),
						)
					}
				}
			}
		}
	}
	if !found {
		t.Error("expected gen_ai.user.message log record")
	}
}
