module github.com/joakimcarlsson/ai/batch

go 1.25.0

require (
	github.com/joakimcarlsson/ai/embeddings v0.2.3
	github.com/joakimcarlsson/ai/llm v0.4.2
	github.com/joakimcarlsson/ai/message v0.3.1
	github.com/joakimcarlsson/ai/tool v0.1.2
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/joakimcarlsson/ai/model v0.6.0 // indirect
	github.com/joakimcarlsson/ai/schema v0.2.0 // indirect
	github.com/joakimcarlsson/ai/tracing v0.1.1 // indirect
	github.com/joakimcarlsson/ai/types v0.1.0 // indirect
	github.com/modelcontextprotocol/go-sdk v1.6.1 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/log v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260618152121-87f3d3e198d3 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260618152121-87f3d3e198d3 // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/joakimcarlsson/ai/embeddings => ../embeddings
	github.com/joakimcarlsson/ai/llm => ../llm
	github.com/joakimcarlsson/ai/message => ../message
	github.com/joakimcarlsson/ai/model => ../model
	github.com/joakimcarlsson/ai/schema => ../schema
	github.com/joakimcarlsson/ai/tool => ../tool
	github.com/joakimcarlsson/ai/tracing => ../tracing
	github.com/joakimcarlsson/ai/types => ../types
)
