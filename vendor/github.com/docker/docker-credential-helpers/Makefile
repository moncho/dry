.PHONY: all deps osxkeychain secretservice test validate wincred

TRAVIS_OS_NAME ?= linux
VERSION = 0.5.0

all: test

deps:
	go get github.com/golang/lint/golint

clean:
	rm -rf bin
	rm -rf release

osxkeychain:
	mkdir bin
	go build -ldflags -s -o bin/docker-credential-osxkeychain osxkeychain/cmd/main_darwin.go

codesign: osxkeychain
	$(eval SIGNINGHASH = $(shell security find-identity -v -p codesigning | grep "Developer ID Application: Docker Inc" | cut -d ' ' -f 4))
	xcrun -log codesign -s $(SIGNINGHASH) --force --verbose bin/docker-credential-osxkeychain
	xcrun codesign --verify --deep --strict --verbose=2 --display bin/docker-credential-osxkeychain

osxrelease: clean test codesign
	mkdir -p release
	cd bin && tar cvfz ../release/docker-credential-osxkeychain-v$(VERSION)-amd64.tar.gz docker-credential-osxkeychain

secretservice:
	mkdir bin
	go build -o bin/docker-credential-secretservice secretservice/cmd/main_linux.go

wincred:
	mkdir bin
	go build -o bin/docker-credential-wincred.exe wincred/cmd/main_windows.go

test:
	# tests all packages except vendor
	go test -v `go list ./... | grep -v /vendor/`

vet: vet_$(TRAVIS_OS_NAME)
	go vet ./credentials

vet_osx:
	go vet ./osxkeychain

vet_linux:
	go vet ./secretservice

validate: vet
	for p in `go list ./... | grep -v /vendor/`; do \
		golint $$p ; \
	done
	gofmt -s -l `ls **/*.go | grep -v vendor`
