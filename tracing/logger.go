package tracing

import (
	"context"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	logger     log.Logger
	loggerOnce sync.Once

	captureContent     bool
	captureContentOnce sync.Once
)

// Logger returns the shared OpenTelemetry logger instance.
func Logger() log.Logger {
	loggerOnce.Do(func() {
		logger = logglobal.GetLoggerProvider().Logger(
			instrumentationName,
		)
	})
	return logger
}

func shouldCaptureContent() bool {
	captureContentOnce.Do(func() {
		val := os.Getenv(
			"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT",
		)
		captureContent = strings.EqualFold(val, "true")
	})
	return captureContent
}

func elideContent(content string) string {
	if shouldCaptureContent() {
		return content
	}
	return "<elided>"
}

func emitLog(ctx context.Context, eventName string, body string) {
	span := oteltrace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	var record log.Record
	record.SetBody(log.StringValue(body))
	record.AddAttributes(log.String("event.name", eventName))
	Logger().Emit(ctx, record)
}

// LogSystemMessage emits a gen_ai.system.message log record.
func LogSystemMessage(ctx context.Context, content string) {
	emitLog(ctx, "gen_ai.system.message", elideContent(content))
}

// LogUserMessage emits a gen_ai.user.message log record.
func LogUserMessage(ctx context.Context, content string) {
	emitLog(ctx, "gen_ai.user.message", elideContent(content))
}

// LogChoice emits a gen_ai.choice log record.
func LogChoice(
	ctx context.Context,
	content string,
	_ string,
) {
	emitLog(ctx, "gen_ai.choice", elideContent(content))
}
