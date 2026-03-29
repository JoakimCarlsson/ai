package tracing

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
		exporter.Reset()
	})
	return exporter
}

func setupMetrics(
	t *testing.T,
) *sdkmetric.ManualReader {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetMeterProvider(prev)
		_ = mp.Shutdown(context.Background())
	})
	return reader
}

type capturedRecord struct {
	Body       otellog.Value
	Attributes []otellog.KeyValue
}

type logCapture struct {
	mu      sync.Mutex
	records []capturedRecord
}

func (c *logCapture) OnEmit(
	_ context.Context,
	record *sdklog.Record,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var attrs []otellog.KeyValue
	record.WalkAttributes(func(kv otellog.KeyValue) bool {
		attrs = append(attrs, kv)
		return true
	})

	c.records = append(c.records, capturedRecord{
		Body:       record.Body(),
		Attributes: attrs,
	})
	return nil
}

func (c *logCapture) Shutdown(context.Context) error {
	return nil
}

func (c *logCapture) ForceFlush(context.Context) error {
	return nil
}

func (c *logCapture) Enabled(
	context.Context,
	sdklog.EnabledParameters,
) bool {
	return true
}

func (c *logCapture) Records() []capturedRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]capturedRecord, len(c.records))
	copy(out, c.records)
	return out
}

func setupLogs(t *testing.T) *logCapture {
	t.Helper()
	capture := &logCapture{}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(capture),
	)
	prev := global.GetLoggerProvider()
	global.SetLoggerProvider(lp)
	t.Cleanup(func() {
		global.SetLoggerProvider(prev)
		_ = lp.Shutdown(context.Background())
	})
	return capture
}

func findSpan(
	spans tracetest.SpanStubs,
	prefix string,
) *tracetest.SpanStub {
	for i, s := range spans {
		if len(s.Name) >= len(prefix) &&
			s.Name[:len(prefix)] == prefix {
			return &spans[i]
		}
	}
	return nil
}

func spanAttr(span *tracetest.SpanStub, key string) string {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			return attr.Value.Emit()
		}
	}
	return ""
}

func spanAttrInt(
	span *tracetest.SpanStub,
	key string,
) int64 {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			return attr.Value.AsInt64()
		}
	}
	return -1
}

func findMetric(
	rm metricdata.ResourceMetrics,
	name string,
) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i, m := range sm.Metrics {
			if m.Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}
