module github.com/joakimcarlsson/ai/memory/sqlite

go 1.26.0

require (
	github.com/joakimcarlsson/ai/message v0.7.0
	github.com/joakimcarlsson/ai/session v0.1.9
)

replace (
	github.com/joakimcarlsson/ai/message => ../../message
	github.com/joakimcarlsson/ai/session => ../../session
)
