module github.com/joakimcarlsson/ai/tokens

go 1.26.0

require (
	github.com/joakimcarlsson/ai/message v0.7.0
	github.com/joakimcarlsson/ai/tool v0.1.5
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/modelcontextprotocol/go-sdk v1.8.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/time v0.16.0 // indirect
)

replace (
	github.com/joakimcarlsson/ai/message => ../message
	github.com/joakimcarlsson/ai/tool => ../tool
)
