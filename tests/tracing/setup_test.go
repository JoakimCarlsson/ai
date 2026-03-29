package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/joakimcarlsson/ai/tracing"
)

func TestSetup_New_WithProcessors(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	providers, err := tracing.New(
		context.Background(),
		tracing.WithSpanProcessors(
			sdktrace.NewSimpleSpanProcessor(exp),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = providers.Shutdown(context.Background()) }()

	ctx, span := tracing.StartGenerateSpan(
		context.Background(),
		"test-model",
		"test-system",
	)
	_ = ctx
	span.End()

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected spans from setup helper")
	}
	if findSpan(spans, "generate_content") == nil {
		t.Error("expected generate_content span")
	}
}

func TestNew_WithResource(t *testing.T) {
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", "test-svc"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	providers, err := tracing.New(
		context.Background(),
		tracing.WithResource(res),
	)
	if err != nil {
		t.Fatal(err)
	}

	if providers.TracerProvider == nil {
		t.Error("expected TracerProvider")
	}
	if providers.MeterProvider == nil {
		t.Error("expected MeterProvider")
	}
	if providers.LoggerProvider == nil {
		t.Error("expected LoggerProvider")
	}

	err = providers.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected shutdown error: %v", err)
	}
}

func TestNew_WithAllOptions(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	reader := sdkmetric.NewManualReader()

	providers, err := tracing.New(
		context.Background(),
		tracing.WithSpanProcessors(
			sdktrace.NewSimpleSpanProcessor(exp),
		),
		tracing.WithMetricReaders(reader),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = providers.Shutdown(context.Background()) }()

	_, span := tracing.StartSpan(
		context.Background(),
		"test-span",
	)
	span.End()

	if len(exp.GetSpans()) == 0 {
		t.Error("expected spans via custom processor")
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
}

func TestProviders_SetGlobal(t *testing.T) {
	prevTP := otel.GetTracerProvider()

	providers, err := tracing.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = providers.Shutdown(context.Background()) }()

	currentTP := otel.GetTracerProvider()

	if currentTP == prevTP {
		t.Error(
			"expected global TracerProvider to change after New()",
		)
	}
}

func TestNew_NoOptions(t *testing.T) {
	providers, err := tracing.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if providers.TracerProvider == nil {
		t.Error("expected TracerProvider")
	}
	if providers.MeterProvider == nil {
		t.Error("expected MeterProvider")
	}
	if providers.LoggerProvider == nil {
		t.Error("expected LoggerProvider")
	}
	if err := providers.Shutdown(context.Background()); err != nil {
		t.Errorf("unexpected shutdown error: %v", err)
	}
}

func TestWithLogProcessors(t *testing.T) {
	capture := &logCapture{}
	providers, err := tracing.New(
		context.Background(),
		tracing.WithLogProcessors(capture),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = providers.Shutdown(context.Background()) }()

	if providers.LoggerProvider == nil {
		t.Error("expected LoggerProvider")
	}
}
