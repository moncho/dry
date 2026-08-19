# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

GOFILES_NOVENDOR := $(shell find . -name vendor -prune -o -type f -name '*.go' -not -name '*.pb.go' -print)
# Populate version variables
PKG := github.com/moncho/dry
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GITCOMMIT := $(shell git rev-parse --short HEAD)
CTIMEVAR=-X $(PKG)/version.GITCOMMIT=$(GITCOMMIT) -X $(PKG)/version.VERSION=$(VERSION)
GO_LDFLAGS=-ldflags "-w $(CTIMEVAR)"
GO_LDFLAGS_STATIC=-ldflags "-w $(CTIMEVAR) -extldflags -static"

print-%: ; @echo $*=$($*)

run: ## Runs dry
	go run ./main.go

build: ## Builds dry
	go build $(GO_LDFLAGS) .

install: ## Installs dry
	go install $(GO_LDFLAGS) $(PKG)

# Keep in sync with .github/workflows/go-lint.yml so local lint predicts CI.
GOLANGCI_LINT_VERSION := v2.10.1
MISSPELL_VERSION := v0.8.0
OK := ✓

lint: ## Runs the same linter as CI (golangci-lint) plus gofmt and misspell
	@echo ">> CODE QUALITY"

	@printf '     GOLANGCI  '
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...; \
	fi
	@printf '%s\n' '$(OK)'

	@printf '     FMT       '
	@out=$$(gofmt -s -l $(GOFILES_NOVENDOR)); if [ -n "$$out" ]; then echo; echo "$$out"; exit 1; fi
	@printf '%s\n' '$(OK)'

	@printf '     SPELL     '
	@if command -v misspell >/dev/null 2>&1; then \
		misspell -error $(GOFILES_NOVENDOR); \
	else \
		go run github.com/golangci/misspell/cmd/misspell@$(MISSPELL_VERSION) -error $(GOFILES_NOVENDOR); \
	fi
	@printf '%s\n' '$(OK)'

fmt: ## Runs fmt
	@gofmt -s -l -w $(GOFILES_NOVENDOR)

test: ## Run tests
	go test -v -cover $(shell go list ./... | grep -v /vendor/ | grep -v mock)

benchmark: ## Run benchmarks
	go test -bench $(shell go list ./... | grep -v /vendor/ | grep -v mock)

.PHONY: help vendor

# Magic as explained here: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

help: ## Shows help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
