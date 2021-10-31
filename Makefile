# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

GOFILES_NOVENDOR := $(shell find . -name vendor -prune -o -type f -name '*.go' -not -name '*.pb.go' -print)
# Populate version variables
# Add to compile time flags
PKG := github.com/moncho/dry
VERSION := $(shell cat APPVERSION)
GITCOMMIT := $(shell git rev-parse --short HEAD)
GITUNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
OS := $(shell uname)
ifneq ($(GITUNTRACKEDCHANGES),)
	GITCOMMIT := $(GITCOMMIT)-dirty
endif
CTIMEVAR=-X $(PKG)/version.GITCOMMIT=$(GITCOMMIT) -X $(PKG)/version.VERSION=$(VERSION)
GO_LDFLAGS=-ldflags "-w $(CTIMEVAR)"
GO_LDFLAGS_STATIC=-ldflags "-w $(CTIMEVAR) -extldflags -static"
GOOSES = darwin freebsd linux windows
GOARCHS = amd64 386 arm arm64
UNSUPPORTED = darwin_arm darwin_386 windows_arm windows_arm64 windows_386 freebsd_arm64 
print-%: ; @echo $*=$($*)

run: ## Runs dry
	GO111MODULE=on go run ./main.go

build: ## Builds dry
	GO111MODULE=on go build .

install: ## Installs dry
	GO111MODULE=on go install $(PKG)

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
	GO111MODULE=on go test -v -cover $(shell go list ./... | grep -v /vendor/ | grep -v mock)

benchmark: ## Run benchmarks
	GO111MODULE=on go test -bench $(shell go list ./... | grep -v /vendor/ | grep -v mock) 

define buildpretty
$(if $(and $(filter-out $(UNSUPPORTED),$(1)_$(2))), \
	mkdir -p ${PREFIX}/cross/$(1)/$(2);
	GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 GO111MODULE=on go build -o ${PREFIX}/cross/$(1)/$(2)/dry -a ${GO_LDFLAGS} .;
)
endef

cross: *.go VERSION ## Cross compiles dry
	$(foreach GOARCH,$(GOARCHS),$(foreach GOOS,$(GOOSES),$(call buildpretty,$(GOOS),$(GOARCH))))

define buildrelease
$(if $(and $(filter-out $(UNSUPPORTED),$(1)_$(2))), \
	mkdir -p ${PREFIX}/cross/$(1)/$(2);
	GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 GO111MODULE=on go build -o ${PREFIX}/cross/dry-$(1)-$(2) -a ${GO_LDFLAGS} .;
)
endef

release: *.go VERSION ## Prepares a dry release
	$(foreach GOARCH,$(GOARCHS),$(foreach GOOS,$(GOOSES),$(call buildrelease,$(GOOS),$(GOARCH))))

clean:
	rm -rf ${PREFIX}/cross

.PHONY: help vendor

# Magic as explained here: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

help: ## Shows help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
