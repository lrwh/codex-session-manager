APP_NAME := csm
MODULE := github.com/liurui/codex-session-manager
VERSION ?= 0.2.0
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: fmt test run build build-linux build-darwin build-windows build-all clean

fmt:
	gofmt -w $$(find . -type f -name '*.go')

test:
	go test ./...

run:
	go run ./cmd/csm

build:
	mkdir -p dist
	go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME) ./cmd/csm

build-linux:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-linux-amd64 ./cmd/csm

build-darwin:
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-darwin-amd64 ./cmd/csm
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-darwin-arm64 ./cmd/csm

build-windows:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-windows-amd64.exe ./cmd/csm

build-all: build-linux build-darwin build-windows

clean:
	rm -rf dist
