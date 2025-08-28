# MCP Integration Example Makefile

.PHONY: help build run server client clean test

# Default target
help:
	@echo "🤖 MCP Integration Example"
	@echo "=========================="
	@echo ""
	@echo "Available commands:"
	@echo "  make build    - Build both server and client"
	@echo "  make run      - Run the complete example (requires OPENAI_API_KEY)"
	@echo "  make server   - Run only the MCP server"
	@echo "  make client   - Run only the client (requires server to be running)"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Clean build artifacts"
	@echo ""
	@echo "Prerequisites:"
	@echo "  export OPENAI_API_KEY=\"your-api-key-here\""

# Build both server and client
build:
	@echo "🔧 Building MCP server..."
	@go build -o mcp_server mcp_server.go
	@echo "🔧 Building MCP client..."
	@go build -o mcp_client main.go
	@echo "✅ Build completed"

# Run the complete example
run: build
	@echo "🚀 Running MCP integration example..."
	@./run.sh

# Run only the MCP server
server: build
	@echo "🔧 Starting MCP server on http://localhost:8080"
	@echo "Press Ctrl+C to stop"
	@./mcp_server

# Run only the client (assumes server is already running)
client: build
	@echo "🤖 Running MCP client..."
	@./mcp_client

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f mcp_server mcp_client
	@rm -f *.txt  # Remove any test files created
	@echo "✅ Clean completed"

# Development targets
dev-server:
	@echo "🔧 Running MCP server in development mode..."
	@go run mcp_server.go

dev-client:
	@echo "🤖 Running MCP client in development mode..."
	@go run main.go

# Check prerequisites
check:
	@echo "🔍 Checking prerequisites..."
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "❌ OPENAI_API_KEY is not set"; \
		echo "Please run: export OPENAI_API_KEY=\"your-api-key-here\""; \
		exit 1; \
	else \
		echo "✅ OPENAI_API_KEY is configured"; \
	fi
	@go version >/dev/null 2>&1 && echo "✅ Go is installed" || (echo "❌ Go is not installed" && exit 1)
	@curl --version >/dev/null 2>&1 && echo "✅ curl is available" || echo "⚠️  curl not found (optional)"