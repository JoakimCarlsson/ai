module github.com/joakimcarlsson/ai/tokens/truncate

go 1.25.0

require (
	github.com/joakimcarlsson/ai/message v0.0.0-00010101000000-000000000000
	github.com/joakimcarlsson/ai/tokens v0.0.0-00010101000000-000000000000
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
	github.com/joakimcarlsson/ai/message => ../../message
	github.com/joakimcarlsson/ai/model => ../../model
	github.com/joakimcarlsson/ai/tokens => ../
	github.com/joakimcarlsson/ai/tool => ../../tool
)
