module github.com/joakimcarlsson/ai/tokens/truncate

go 1.25.0

require (
	github.com/joakimcarlsson/ai/message v0.2.0
	github.com/joakimcarlsson/ai/tokens v0.2.1
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/joakimcarlsson/ai/model v0.5.0 // indirect
	github.com/joakimcarlsson/ai/tool v0.1.2 // indirect
	github.com/modelcontextprotocol/go-sdk v1.6.1 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace (
	github.com/joakimcarlsson/ai/message => ../../message
	github.com/joakimcarlsson/ai/model => ../../model
	github.com/joakimcarlsson/ai/tokens => ../
	github.com/joakimcarlsson/ai/tool => ../../tool
)
