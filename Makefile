.PHONY: install fmt format lint test release-tag release-publish llms

GOPATH_FWD := $(subst \,/,$(shell go env GOPATH))

ifeq ($(OS),Windows_NT)
    GOLANGCI := cmd /c "set GOTOOLCHAIN=local&& golangci-lint run ./..."
else
    GOLANGCI := GOTOOLCHAIN=local $(GOPATH_FWD)/bin/golangci-lint run ./...
endif

install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/segmentio/golines@latest

fmt:
	$(GOPATH_FWD)/bin/goimports -w .
	$(GOPATH_FWD)/bin/golines -m 80 -w .

lint:
	go vet ./...
	$(GOLANGCI)

test:
	go test -short ./...

release-tag:
	@scripts/release.sh tag -m $(MODULE) -v $(VERSION) --push

release-publish:
	@scripts/release.sh release --publish

llms:
	cd cmd/llmstxt && go run . -config ../../www/mkdocs.yml -docs ../../www/docs -out ../../www/docs
