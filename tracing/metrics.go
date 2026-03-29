package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Meter returns the OpenTelemetry meter instance.
func Meter() metric.Meter {
	return otel.Meter(instrumentationName)
}

func getOperationDuration() metric.Float64Histogram {
	h, _ := Meter().Float64Histogram(
		"gen_ai.client.operation.duration",
		metric.WithDescription(
			"Duration of AI provider operations",
		),
		metric.WithUnit("s"),
	)
	return h
}

func getTokenUsage() metric.Int64Counter {
	c, _ := Meter().Int64Counter(
		"gen_ai.client.token.usage",
		metric.WithDescription(
			"Token consumption by AI provider operations",
		),
		metric.WithUnit("{token}"),
	)
	return c
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
	getOperationDuration().Record(ctx, duration.Seconds(), opt)

	if inputTokens > 0 {
		getTokenUsage().Add(ctx, inputTokens, metric.WithAttributes(
			append(
				attrs,
				attribute.String("gen_ai.token.type", "input"),
			)...,
		))
	}
	if outputTokens > 0 {
		getTokenUsage().Add(ctx, outputTokens, metric.WithAttributes(
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
