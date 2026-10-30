module github.com/joakimcarlsson/ai/examples/agent/basic

go 1.26.0

require (
	github.com/joakimcarlsson/ai/agent v0.5.8
	github.com/joakimcarlsson/ai/llm/openai v0.15.0
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.33.0 // indirect
	github.com/joakimcarlsson/ai/embeddings v0.3.1 // indirect
	github.com/joakimcarlsson/ai/llm v0.8.0 // indirect
	github.com/joakimcarlsson/ai/memory v0.2.14 // indirect
	github.com/joakimcarlsson/ai/message v0.7.0 // indirect
	github.com/joakimcarlsson/ai/prompt v0.1.1 // indirect
	github.com/joakimcarlsson/ai/schema v0.2.2 // indirect
	github.com/joakimcarlsson/ai/session v0.1.9 // indirect
	github.com/joakimcarlsson/ai/tokens v0.2.10 // indirect
	github.com/joakimcarlsson/ai/tool v0.1.4 // indirect
	github.com/joakimcarlsson/ai/tracing v0.2.2 // indirect
	github.com/joakimcarlsson/ai/types v0.2.1 // indirect
	github.com/modelcontextprotocol/go-sdk v1.8.0 // indirect
	github.com/openai/openai-go/v3 v3.90.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/tidwall/gjson v1.20.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.2 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/log v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.1 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20261026165753-2ffab32bce7e // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261026165753-2ffab32bce7e // indirect
	google.golang.org/grpc v1.85.0 // indirect
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
	github.com/joakimcarlsson/ai/llm/anthropic => ../../../llm/anthropic
	github.com/joakimcarlsson/ai/llm/gemini => ../../../llm/gemini
	github.com/joakimcarlsson/ai/llm/openai => ../../../llm/openai
	github.com/joakimcarlsson/ai/memory => ../../../memory
	github.com/joakimcarlsson/ai/message => ../../../message
	github.com/joakimcarlsson/ai/prompt => ../../../prompt
	github.com/joakimcarlsson/ai/rerankers => ../../../rerankers
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
