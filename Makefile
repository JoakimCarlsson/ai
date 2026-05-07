.PHONY: install fmt lint test test-integration build modules release-tag release release-warm llms workspace

GOPATH_FWD := $(subst \,/,$(shell go env GOPATH))

ifeq ($(OS),Windows_NT)
    GOLANGCI := cmd /c "set GOTOOLCHAIN=local&& golangci-lint run ./..."
else
    GOLANGCI := GOTOOLCHAIN=local $(GOPATH_FWD)/bin/golangci-lint run ./...
    MODULES := $(shell scripts/release.sh modules 2>/dev/null)
endif

install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/segmentio/golines@latest

workspace:
ifeq ($(OS),Windows_NT)
	@if not exist go.work copy go.work.example go.work >NUL
else
	@test -f go.work || cp go.work.example go.work
endif

fmt:
	$(GOPATH_FWD)/bin/goimports -w .
	$(GOPATH_FWD)/bin/golines -m 80 -w .

lint: workspace
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference = 'Stop'; $$modules = (go work edit -json go.work.example | ConvertFrom-Json).Use.DiskPath; foreach ($$module in $$modules) { $$dir = $$module -replace '^\./',''; Write-Host ('==> lint ' + $$dir); Push-Location $$dir; try { go vet ./...; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE }; $$env:GOTOOLCHAIN = 'local'; golangci-lint run ./...; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE } } finally { Pop-Location } }"
else
	@for dir in $(MODULES); do \
		echo "==> lint $$dir"; \
		(cd "$$dir" && go vet ./... && $(GOLANGCI)) || exit 1; \
	done
endif

build: workspace
	@for dir in $(MODULES); do \
		echo "==> build $$dir"; \
		(cd "$$dir" && go build ./...) || exit 1; \
	done

test: workspace
	@for dir in $(MODULES); do \
		case "$$dir" in tests|tests/*|cmd/*) continue ;; esac; \
		echo "==> test $$dir"; \
		(cd "$$dir" && go test -short ./...) || exit 1; \
	done

test-integration: workspace
	cd tests && go test -timeout 300s ./...

modules:
	@scripts/release.sh modules

release-tag:
	@scripts/release.sh tag -m $(MODULE) -v $(VERSION) --push

release:
	@if [ "$(PUBLISH)" = "1" ]; then \
		scripts/release.sh release --publish; \
	else \
		scripts/release.sh release; \
	fi

release-warm:
	@scripts/release.sh warm -t $(TAG)

llms:
	cd cmd/llmstxt && go run . -config ../../www/mkdocs.yml -docs ../../www/docs -out ../../www/docs
