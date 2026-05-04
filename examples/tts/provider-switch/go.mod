module github.com/joakimcarlsson/ai/examples/tts/provider-switch

go 1.25.0

require (
	github.com/joakimcarlsson/ai/model v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/tts v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/tts/elevenlabs v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/tts/openai v0.0.0-00010101000000-000000000000
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/joakimcarlsson/ai/tracing v0.0.0-00010101000000-000000000000 // indirect
	github.com/openai/openai-go v1.12.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0 // indirect
	go.opentelemetry.io/otel/log v0.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.19.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/joakimcarlsson/ai/agent => ../../../agent
	github.com/joakimcarlsson/ai/agent/memory => ../../../agent/memory
	github.com/joakimcarlsson/ai/batch => ../../../batch
	github.com/joakimcarlsson/ai/batch/concurrent => ../../../batch/concurrent
	github.com/joakimcarlsson/ai/embeddings => ../../../embeddings
	github.com/joakimcarlsson/ai/embeddings/cohere => ../../../embeddings/cohere
	github.com/joakimcarlsson/ai/embeddings/openai => ../../../embeddings/openai
	github.com/joakimcarlsson/ai/embeddings/voyage => ../../../embeddings/voyage
	github.com/joakimcarlsson/ai/fim => ../../../fim
	github.com/joakimcarlsson/ai/fim/deepseek => ../../../fim/deepseek
	github.com/joakimcarlsson/ai/fim/mistral => ../../../fim/mistral
	github.com/joakimcarlsson/ai/image => ../../../image
	github.com/joakimcarlsson/ai/image/gemini => ../../../image/gemini
	github.com/joakimcarlsson/ai/image/openai => ../../../image/openai
	github.com/joakimcarlsson/ai/llm => ../../../llm
	github.com/joakimcarlsson/ai/llm/anthropic => ../../../llm/anthropic
	github.com/joakimcarlsson/ai/llm/gemini => ../../../llm/gemini
	github.com/joakimcarlsson/ai/llm/openai => ../../../llm/openai
	github.com/joakimcarlsson/ai/message => ../../../message
	github.com/joakimcarlsson/ai/model => ../../../model
	github.com/joakimcarlsson/ai/prompt => ../../../prompt
	github.com/joakimcarlsson/ai/rerankers => ../../../rerankers
	github.com/joakimcarlsson/ai/rerankers/cohere => ../../../rerankers/cohere
	github.com/joakimcarlsson/ai/rerankers/voyage => ../../../rerankers/voyage
	github.com/joakimcarlsson/ai/schema => ../../../schema
	github.com/joakimcarlsson/ai/stt => ../../../stt
	github.com/joakimcarlsson/ai/stt/deepgram => ../../../stt/deepgram
	github.com/joakimcarlsson/ai/stt/openai => ../../../stt/openai
	github.com/joakimcarlsson/ai/tokens => ../../../tokens
	github.com/joakimcarlsson/ai/tokens/truncate => ../../../tokens/truncate
	github.com/joakimcarlsson/ai/tool => ../../../tool
	github.com/joakimcarlsson/ai/tracing => ../../../tracing
	github.com/joakimcarlsson/ai/tts => ../../../tts
	github.com/joakimcarlsson/ai/tts/elevenlabs => ../../../tts/elevenlabs
	github.com/joakimcarlsson/ai/tts/openai => ../../../tts/openai
	github.com/joakimcarlsson/ai/types => ../../../types
)
