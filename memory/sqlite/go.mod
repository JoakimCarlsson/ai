module github.com/joakimcarlsson/ai/memory/sqlite

go 1.25.0

require (
	github.com/joakimcarlsson/ai/message v0.5.2
	github.com/joakimcarlsson/ai/session v0.1.6
)

replace (
	github.com/joakimcarlsson/ai/message => ../../message
	github.com/joakimcarlsson/ai/session => ../../session
)
