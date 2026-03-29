package tracing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/joakimcarlsson/ai/tracing"
)

func TestRecordMetrics_Duration(t *testing.T) {
	reader := setupMetrics(t)

	tracing.RecordMetrics(
		context.Background(),
		"generate_content",
		"gpt-4",
		"openai",
		50*time.Millisecond,
		10,
		5,
		nil,
	)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	m := findMetric(rm, "gen_ai.client.operation.duration")
	if m == nil {
		t.Fatal("expected gen_ai.client.operation.duration metric")
	}
}

func TestRecordMetrics_TokenUsage(t *testing.T) {
	reader := setupMetrics(t)

	tracing.RecordMetrics(
		context.Background(),
		"generate_content",
		"gpt-4",
		"openai",
		50*time.Millisecond,
		100,
		50,
		nil,
	)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	m := findMetric(rm, "gen_ai.client.token.usage")
	if m == nil {
		t.Fatal("expected gen_ai.client.token.usage metric")
	}
}

func TestRecordMetrics_ErrorAttr(t *testing.T) {
	reader := setupMetrics(t)

	tracing.RecordMetrics(
		context.Background(),
		"generate_content",
		"gpt-4",
		"openai",
		10*time.Millisecond,
		0,
		0,
		fmt.Errorf("connection refused"),
	)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	m := findMetric(rm, "gen_ai.client.operation.duration")
	if m == nil {
		t.Fatal("expected gen_ai.client.operation.duration metric")
	}
}

func TestRecordMetrics_NoTokensWhenZero(t *testing.T) {
	reader := setupMetrics(t)

	tracing.RecordMetrics(
		context.Background(),
		"generate_audio",
		"eleven-turbo",
		"elevenlabs",
		10*time.Millisecond,
		0,
		0,
		nil,
	)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	m := findMetric(rm, "gen_ai.client.token.usage")
	if m != nil {
		t.Error("expected no token.usage metric when tokens are zero")
	}
}
