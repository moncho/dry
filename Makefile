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

lint: ## Runs linters
	@echo ">> CODE QUALITY"

	@echo -n "     REVIVE    "
	@which revive > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/mgechev/revive; \
	fi
	@revive -formatter friendly -exclude vendor/... ./...
	@printf '%s\n' '$(OK)'

	@echo -n "     FMT       "
	@$(foreach gofile, $(GOFILES_NOVENDOR),\
			out=$$(gofmt -s -l -d -e $(gofile) | tee /dev/stderr); if [ -n "$$out" ]; then exit 1; fi;)
	@printf '%s\n' '$(OK)'

	@echo -n "     SPELL     "
	@which misspell > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/client9/misspell/cmd/misspell; \
	fi
	@$(foreach gofile, $(GOFILES_NOVENDOR),\
			misspell --error $(gofile) || exit 1;)
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
