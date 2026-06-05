module github.com/joakimcarlsson/ai/eval/judge

go 1.25.0

require (
	github.com/joakimcarlsson/ai/eval v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/llm v0.1.0
	github.com/joakimcarlsson/ai/message v0.1.0
)

replace (
	github.com/joakimcarlsson/ai/eval => ../
	github.com/joakimcarlsson/ai/llm => ../../llm
	github.com/joakimcarlsson/ai/message => ../../message
	github.com/joakimcarlsson/ai/model => ../../model
	github.com/joakimcarlsson/ai/schema => ../../schema
	github.com/joakimcarlsson/ai/tool => ../../tool
	github.com/joakimcarlsson/ai/tracing => ../../tracing
	github.com/joakimcarlsson/ai/types => ../../types
)
