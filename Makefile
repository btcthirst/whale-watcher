.PHONY: help deps build run test test-cover vet lint clean fmt

# === Variables ===
BINARY_NAME=whale-watcher
MAIN_PATH=./cmd/watcher
GO ?= go
GOLANGCI_LINT ?= golangci-lint
VERSION=$(shell git describe --tags 2>/dev/null || echo 'dev')
BUILD_FLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

# === Default Target ===
default: help

## help: Print this help message
help:
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  \033[36m%-15s\033[0m %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

## deps: Download and tidy Go dependencies
deps:
	@echo "📦 Tidying and downloading dependencies..."
	$(GO) mod tidy
	$(GO) mod download

## build: Build the application binary
build: deps
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GO) build $(BUILD_FLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build complete: bin/$(BINARY_NAME)"

## run: Run the application directly
run: deps
	@echo "🚀 Running $(BINARY_NAME)..."
	$(GO) run $(MAIN_PATH)/main.go

## test: Run unit tests with race detector
test:
	@echo "🧪 Running tests..."
	$(GO) test -race -v ./...

## test-cover: Run tests and generate coverage report
test-cover:
	@echo "📊 Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated at coverage.html"

## vet: Run go vet to examine Go source code
vet:
	@echo "🔍 Running go vet..."
	$(GO) vet ./...

## lint: Run golangci-lint
lint:
	@echo "🧹 Running linter..."
	$(GOLANGCI_LINT) run ./...

## fmt: Format Go source code
fmt:
	@echo "🎨 Formatting code..."
	$(GO) fmt ./...

## clean: Remove build artifacts and temporary files
clean:
	@echo "🗑️  Cleaning..."
	rm -rf bin/ data/ logs/ coverage.out coverage.html
	@echo "✅ Clean complete"