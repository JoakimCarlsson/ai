module github.com/joakimcarlsson/ai/llm/azure

go 1.25.0

require (
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
	github.com/joakimcarlsson/ai/llm v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/llm/openai v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/model v0.0.0-00010101000000-000000000000
	github.com/openai/openai-go v1.12.0
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.20.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/joakimcarlsson/ai/message v0.0.0-00010101000000-000000000000 // indirect
	github.com/joakimcarlsson/ai/schema v0.0.0-00010101000000-000000000000 // indirect
	github.com/joakimcarlsson/ai/tool v0.0.0-00010101000000-000000000000 // indirect
	github.com/joakimcarlsson/ai/tracing v0.0.0-00010101000000-000000000000 // indirect
	github.com/joakimcarlsson/ai/types v0.0.0-00010101000000-000000000000 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/modelcontextprotocol/go-sdk v1.6.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
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
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/joakimcarlsson/ai/llm => ../
	github.com/joakimcarlsson/ai/llm/openai => ../openai
	github.com/joakimcarlsson/ai/message => ../../message
	github.com/joakimcarlsson/ai/model => ../../model
	github.com/joakimcarlsson/ai/schema => ../../schema
	github.com/joakimcarlsson/ai/tool => ../../tool
	github.com/joakimcarlsson/ai/tracing => ../../tracing
	github.com/joakimcarlsson/ai/types => ../../types
)
