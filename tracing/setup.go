package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Providers holds the initialized OpenTelemetry providers.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// Shutdown gracefully shuts down all providers.
func (p *Providers) Shutdown(ctx context.Context) error {
	var firstErr error
	if p.TracerProvider != nil {
		if err := p.TracerProvider.Shutdown(ctx); err != nil &&
			firstErr == nil {
			firstErr = err
		}
	}
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if p.LoggerProvider != nil {
		if err := p.LoggerProvider.Shutdown(ctx); err != nil &&
			firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SetGlobal registers all providers as the global OpenTelemetry providers.
func (p *Providers) SetGlobal() {
	if p.TracerProvider != nil {
		otel.SetTracerProvider(p.TracerProvider)
	}
	if p.MeterProvider != nil {
		otel.SetMeterProvider(p.MeterProvider)
	}
	if p.LoggerProvider != nil {
		logglobal.SetLoggerProvider(p.LoggerProvider)
	}
}

type config struct {
	resource       *resource.Resource
	otlpEndpoint   string
	spanProcessors []sdktrace.SpanProcessor
	metricReaders  []sdkmetric.Reader
	logProcessors  []sdklog.Processor
}

// Option configures the tracing setup.
type Option func(*config)

// WithResource sets the OpenTelemetry resource for all providers.
func WithResource(r *resource.Resource) Option {
	return func(c *config) {
		c.resource = r
	}
}

// WithOTLPEndpoint configures OTLP HTTP exporters for traces, metrics, and logs.
func WithOTLPEndpoint(endpoint string) Option {
	return func(c *config) {
		c.otlpEndpoint = endpoint
	}
}

// WithSpanProcessors registers additional span processors.
func WithSpanProcessors(
	processors ...sdktrace.SpanProcessor,
) Option {
	return func(c *config) {
		c.spanProcessors = append(
			c.spanProcessors,
			processors...,
		)
	}
}

// WithMetricReaders registers additional metric readers.
func WithMetricReaders(readers ...sdkmetric.Reader) Option {
	return func(c *config) {
		c.metricReaders = append(c.metricReaders, readers...)
	}
}

// WithLogProcessors registers additional log processors.
func WithLogProcessors(
	processors ...sdklog.Processor,
) Option {
	return func(c *config) {
		c.logProcessors = append(
			c.logProcessors,
			processors...,
		)
	}
}

// New creates and globally registers OpenTelemetry providers.
func New(
	ctx context.Context,
	opts ...Option,
) (*Providers, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	endpoint := cfg.otlpEndpoint
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	if endpoint != "" {
		if err := configureOTLP(
			ctx,
			cfg,
			endpoint,
		); err != nil {
			return nil, err
		}
	}

	tpOpts := []sdktrace.TracerProviderOption{}
	if cfg.resource != nil {
		tpOpts = append(tpOpts, sdktrace.WithResource(cfg.resource))
	}
	for _, sp := range cfg.spanProcessors {
		tpOpts = append(tpOpts, sdktrace.WithSpanProcessor(sp))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)

	mpOpts := []sdkmetric.Option{}
	if cfg.resource != nil {
		mpOpts = append(mpOpts, sdkmetric.WithResource(cfg.resource))
	}
	for _, r := range cfg.metricReaders {
		mpOpts = append(mpOpts, sdkmetric.WithReader(r))
	}
	mp := sdkmetric.NewMeterProvider(mpOpts...)

	lpOpts := []sdklog.LoggerProviderOption{}
	if cfg.resource != nil {
		lpOpts = append(lpOpts, sdklog.WithResource(cfg.resource))
	}
	for _, p := range cfg.logProcessors {
		lpOpts = append(lpOpts, sdklog.WithProcessor(p))
	}
	lp := sdklog.NewLoggerProvider(lpOpts...)

	providers := &Providers{
		TracerProvider: tp,
		MeterProvider:  mp,
		LoggerProvider: lp,
	}
	providers.SetGlobal()

	return providers, nil
}

func configureOTLP(
	ctx context.Context,
	cfg *config,
	endpoint string,
) error {
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
	)
	if err != nil {
		return err
	}
	cfg.spanProcessors = append(
		cfg.spanProcessors,
		sdktrace.NewBatchSpanProcessor(traceExporter),
	)

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
	)
	if err != nil {
		return err
	}
	cfg.metricReaders = append(
		cfg.metricReaders,
		sdkmetric.NewPeriodicReader(metricExporter),
	)

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(endpoint),
	)
	if err != nil {
		return err
	}
	cfg.logProcessors = append(
		cfg.logProcessors,
		sdklog.NewBatchProcessor(logExporter),
	)

	return nil
}
