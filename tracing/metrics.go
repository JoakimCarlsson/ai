package tracing

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter     metric.Meter
	meterOnce sync.Once

	operationDuration metric.Float64Histogram
	tokenUsage        metric.Int64Counter
)

// Meter returns the shared OpenTelemetry meter instance.
func Meter() metric.Meter {
	meterOnce.Do(func() {
		meter = otel.Meter(instrumentationName)

		operationDuration, _ = meter.Float64Histogram(
			"gen_ai.client.operation.duration",
			metric.WithDescription(
				"Duration of AI provider operations",
			),
			metric.WithUnit("s"),
		)

		tokenUsage, _ = meter.Int64Counter(
			"gen_ai.client.token.usage",
			metric.WithDescription(
				"Token consumption by AI provider operations",
			),
			metric.WithUnit("{token}"),
		)
	})
	return meter
}

func initMetrics() {
	Meter()
}

// RecordMetrics records operation duration and token usage metrics.
func RecordMetrics(
	ctx context.Context,
	operation string,
	modelName string,
	system string,
	duration time.Duration,
	inputTokens int64,
	outputTokens int64,
	err error,
) {
	initMetrics()

	attrs := []attribute.KeyValue{
		AttrOperationName.String(operation),
		AttrSystem.String(system),
		AttrRequestModel.String(modelName),
	}

	if err != nil {
		attrs = append(
			attrs,
			attribute.String("error.type", errorType(err)),
		)
	}

	opt := metric.WithAttributes(attrs...)
	operationDuration.Record(ctx, duration.Seconds(), opt)

	if inputTokens > 0 {
		tokenUsage.Add(ctx, inputTokens, metric.WithAttributes(
			append(
				attrs,
				attribute.String("gen_ai.token.type", "input"),
			)...,
		))
	}
	if outputTokens > 0 {
		tokenUsage.Add(ctx, outputTokens, metric.WithAttributes(
			append(
				attrs,
				attribute.String("gen_ai.token.type", "output"),
			)...,
		))
	}
}

func errorType(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
