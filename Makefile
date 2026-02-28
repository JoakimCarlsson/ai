lint:
	go vet ./...
	$(shell go env GOPATH)/bin/golangci-lint run ./...

fmt:
	$(shell go env GOPATH)/bin/goimports -w .
	$(shell go env GOPATH)/bin/golines -m 80 -w .
	cd web && bun prettier --write "src/**/*.{ts,tsx}"
	cd agent && $(DENO) fmt
	cd sidecar && $(DENO) fmt