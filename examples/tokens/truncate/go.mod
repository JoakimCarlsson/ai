module github.com/joakimcarlsson/ai/examples/tokens/truncate

go 1.25.0

require (
	github.com/joakimcarlsson/ai/message v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/tokens v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/tokens/truncate v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/joakimcarlsson/ai/model v0.0.0-00010101000000-000000000000 // indirect
	github.com/joakimcarlsson/ai/tool v0.0.0-00010101000000-000000000000 // indirect
	github.com/modelcontextprotocol/go-sdk v1.6.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace (
	github.com/joakimcarlsson/ai/agent => ../../../agent
	github.com/joakimcarlsson/ai/memory => ../../../memory
	github.com/joakimcarlsson/ai/batch => ../../../batch
	github.com/joakimcarlsson/ai/batch/concurrent => ../../../batch/concurrent
	github.com/joakimcarlsson/ai/embeddings => ../../../embeddings
	github.com/joakimcarlsson/ai/fim => ../../../fim
	github.com/joakimcarlsson/ai/image => ../../../image
	github.com/joakimcarlsson/ai/llm => ../../../llm
	github.com/joakimcarlsson/ai/message => ../../../message
	github.com/joakimcarlsson/ai/model => ../../../model
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
