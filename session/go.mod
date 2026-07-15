module github.com/joakimcarlsson/ai/session

go 1.25.0

require github.com/joakimcarlsson/ai/message v0.4.0

require github.com/joakimcarlsson/ai/model v0.6.0 // indirect

replace (
	github.com/joakimcarlsson/ai/message => ../message
	github.com/joakimcarlsson/ai/model => ../model
)
