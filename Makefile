.PHONY: build run test clean fmt lint deps

BINARY=whale-watcher
BUILD_FLAGS=-ldflags="-s -w -X main.Version=$(shell git describe --tags 2>/dev/null || echo 'dev')"

deps:
	go mod tidy

build: deps
	@echo "🔨 Building..."
	go build $(BUILD_FLAGS) -o bin/$(BINARY) ./cmd/watcher

run: deps
	@echo "🚀 Running..."
	go run ./cmd/watcher/main.go

test:
	go test -race -cover ./...

clean:
	rm -rf bin/ data/ logs/

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...