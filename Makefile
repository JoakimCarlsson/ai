.PHONY: install fmt lint test

install:
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/segmentio/golines@latest

fmt:
	$(shell go env GOPATH)/bin/goimports -w .
	$(shell go env GOPATH)/bin/golines -m 80 -w .

lint:
	go vet ./...
	$(shell go env GOPATH)/bin/golangci-lint run ./...

test:
	go test -short ./...
