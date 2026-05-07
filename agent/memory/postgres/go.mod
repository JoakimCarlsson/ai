module github.com/joakimcarlsson/ai/agent/memory/postgres

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/joakimcarlsson/ai/agent v0.1.0
	github.com/joakimcarlsson/ai/message v0.1.0
	github.com/lib/pq v1.12.3
)

require github.com/joakimcarlsson/ai/model v0.1.0 // indirect

replace (
	github.com/joakimcarlsson/ai/agent => ../../
	github.com/joakimcarlsson/ai/agent/memory => ../
	github.com/joakimcarlsson/ai/embeddings => ../../../embeddings
	github.com/joakimcarlsson/ai/llm => ../../../llm
	github.com/joakimcarlsson/ai/message => ../../../message
	github.com/joakimcarlsson/ai/model => ../../../model
	github.com/joakimcarlsson/ai/prompt => ../../../prompt
	github.com/joakimcarlsson/ai/schema => ../../../schema
	github.com/joakimcarlsson/ai/tokens => ../../../tokens
	github.com/joakimcarlsson/ai/tool => ../../../tool
	github.com/joakimcarlsson/ai/tracing => ../../../tracing
	github.com/joakimcarlsson/ai/types => ../../../types
)
