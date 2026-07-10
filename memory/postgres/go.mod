module github.com/joakimcarlsson/ai/memory/postgres

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/joakimcarlsson/ai/message v0.3.1
	github.com/joakimcarlsson/ai/session v0.1.2
	github.com/lib/pq v1.12.3
)

require github.com/joakimcarlsson/ai/model v0.6.0 // indirect

replace (
	github.com/joakimcarlsson/ai/message => ../../message
	github.com/joakimcarlsson/ai/model => ../../model
	github.com/joakimcarlsson/ai/session => ../../session
)
