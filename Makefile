# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help build test run-example clean fmt vet tidy env

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

env: ## Export environment variables from .env file
	@echo "Exporting environment variables from .env..."
	@set -a && source .env && set +a
	@echo "Environment variables loaded. Use 'source .env' or run commands with 'make run-example'"

build: ## Build all packages
	@echo "Building all packages..."
	go build ./...

test: ## Run tests
	@echo "Running tests..."
	go test ./...

run-example: ## Run the basic example with environment variables from .env
	@echo "Running basic example with API keys from .env..."
	@cd examples/basic && \
	export $$(cat ../../.env | xargs) && \
	go run main.go

run-example-verbose: ## Run the basic example with verbose output
	@echo "Running basic example with verbose tracing..."
	@cd examples/basic && \
	export $$(cat ../../.env | xargs) && \
	TRACE=1 go run main.go

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	go clean ./...

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	go mod tidy

lint: ## Run linting (requires golangci-lint)
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

check: fmt vet tidy ## Run all checks (format, vet, tidy)
	@echo "All checks completed successfully!"

install-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Development workflow
dev: check test build ## Full development workflow (check, test, build)
	@echo "Development workflow completed successfully!"

# Demo commands
demo: run-example ## Alias for run-example

demo-no-keys: ## Run example without API keys (placeholder mode)
	@echo "Running basic example without API keys (placeholder mode)..."
	@cd examples/basic && go run main.go