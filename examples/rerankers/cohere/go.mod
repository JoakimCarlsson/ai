module github.com/joakimcarlsson/ai/examples/rerankers/cohere

go 1.25.0

require (
	github.com/joakimcarlsson/ai/model v0.8.0
	github.com/joakimcarlsson/ai/rerankers/cohere v0.1.5
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0 // indirect
	github.com/joakimcarlsson/ai/rerankers v0.2.3 // indirect
	github.com/joakimcarlsson/ai/tracing v0.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.21.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.45.0 // indirect
	go.opentelemetry.io/otel/log v0.21.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.21.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260810153831-ec0a7760b754 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260810153831-ec0a7760b754 // indirect
	google.golang.org/grpc v1.83.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	github.com/joakimcarlsson/ai/agent => ../../../agent
	github.com/joakimcarlsson/ai/batch => ../../../batch
	github.com/joakimcarlsson/ai/batch/concurrent => ../../../batch/concurrent
	github.com/joakimcarlsson/ai/embeddings => ../../../embeddings
	github.com/joakimcarlsson/ai/fim => ../../../fim
	github.com/joakimcarlsson/ai/image => ../../../image
	github.com/joakimcarlsson/ai/llm => ../../../llm
	github.com/joakimcarlsson/ai/memory => ../../../memory
	github.com/joakimcarlsson/ai/message => ../../../message
	github.com/joakimcarlsson/ai/model => ../../../model
	github.com/joakimcarlsson/ai/prompt => ../../../prompt
	github.com/joakimcarlsson/ai/rerankers => ../../../rerankers
	github.com/joakimcarlsson/ai/rerankers/cohere => ../../../rerankers/cohere
	github.com/joakimcarlsson/ai/schema => ../../../schema
	github.com/joakimcarlsson/ai/session => ../../../session
	github.com/joakimcarlsson/ai/stt => ../../../stt
	github.com/joakimcarlsson/ai/tokens => ../../../tokens
	github.com/joakimcarlsson/ai/tokens/truncate => ../../../tokens/truncate
	github.com/joakimcarlsson/ai/tool => ../../../tool
	github.com/joakimcarlsson/ai/tracing => ../../../tracing
	github.com/joakimcarlsson/ai/tts => ../../../tts
	github.com/joakimcarlsson/ai/types => ../../../types
)
