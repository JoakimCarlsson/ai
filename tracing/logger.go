package tracing

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Logger returns the OpenTelemetry logger instance.
func Logger() log.Logger {
	return logglobal.GetLoggerProvider().Logger(instrumentationName)
}

func shouldCaptureContent() bool {
	val := os.Getenv(
		"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
	)
	return strings.EqualFold(val, "true")
}

func elideContent(content string) string {
	if shouldCaptureContent() {
		return content
	}
	return "<elided>"
}

func emitLog(ctx context.Context, eventName string, body log.Value) {
	span := oteltrace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	var record log.Record
	record.SetBody(body)
	record.AddAttributes(log.String("event.name", eventName))
	Logger().Emit(ctx, record)
}

// LogSystemMessage emits a gen_ai.system.message log record.
func LogSystemMessage(ctx context.Context, content string) {
	emitLog(ctx, "gen_ai.system.message", log.MapValue(
		log.String("content", elideContent(content)),
	))
}

// LogUserMessage emits a gen_ai.user.message log record.
func LogUserMessage(ctx context.Context, content string) {
	emitLog(ctx, "gen_ai.user.message", log.MapValue(
		log.String("content", elideContent(content)),
	))
}

// LogChoice emits a gen_ai.choice log record.
func LogChoice(
	ctx context.Context,
	content string,
	finishReason string,
) {
	emitLog(ctx, "gen_ai.choice", log.MapValue(
		log.Int("index", 0),
		log.String("content", elideContent(content)),
		log.String("finish_reason", finishReason),
	))
}
