module github.com/joakimcarlsson/ai/memory/sqlite/tests

go 1.26.0

replace github.com/joakimcarlsson/ai/memory/sqlite => ../

replace github.com/joakimcarlsson/ai/message => ../../../message

replace github.com/joakimcarlsson/ai/session => ../../../session

require (
	github.com/joakimcarlsson/ai/memory/sqlite v0.1.9
	github.com/joakimcarlsson/ai/message v0.5.2
	github.com/stretchr/testify v1.12.1
	modernc.org/sqlite v1.58.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/joakimcarlsson/ai/session v0.1.6 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.48.0 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)
