#!/bin/bash

# MCP Integration Example Runner
# This script helps you run the MCP integration example

set -e

echo "🤖 MCP Integration Example Setup"
echo "================================"
echo

# Check if OPENAI_API_KEY is set
if [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ Error: OPENAI_API_KEY environment variable is not set"
    echo "Please set your OpenAI API key:"
    echo "  export OPENAI_API_KEY=\"your-api-key-here\""
    echo
    exit 1
fi

echo "✅ OpenAI API key is configured"
echo

# Function to check if a port is available
check_port() {
    local port=$1
    if lsof -i :$port >/dev/null 2>&1; then
        return 1
    else
        return 0
    fi
}

# Check if port 8080 is available
if ! check_port 8080; then
    echo "❌ Error: Port 8080 is already in use"
    echo "Please stop any service using port 8080 or modify the server port"
    echo
    exit 1
fi

echo "✅ Port 8080 is available"
echo

# Build the applications
echo "🔧 Building applications..."
go build -o mcp_server mcp_server.go
go build -o mcp_client main.go
echo "✅ Build completed"
echo

# Start the MCP server in the background
echo "🚀 Starting MCP server..."
./mcp_server &
MCP_SERVER_PID=$!

# Function to cleanup on exit
cleanup() {
    echo
    echo "🧹 Cleaning up..."
    if [ ! -z "$MCP_SERVER_PID" ]; then
        kill $MCP_SERVER_PID 2>/dev/null || true
        wait $MCP_SERVER_PID 2>/dev/null || true
    fi
    echo "✅ Cleanup completed"
}

# Set trap to cleanup on script exit
trap cleanup EXIT

# Wait for server to start
echo "⏳ Waiting for MCP server to start..."
sleep 3

# Check if server is running
if ! curl -s http://localhost:8080 >/dev/null 2>&1; then
    echo "❌ Error: MCP server failed to start"
    exit 1
fi

echo "✅ MCP server is running"
echo

# Run the client
echo "🤖 Running MCP client example..."
echo "================================"
echo
./mcp_client

echo
echo "🎉 Example completed successfully!"
echo
echo "To run again:"
echo "  ./run.sh"
echo
echo "To run components separately:"
echo "  # Terminal 1:"
echo "  go run mcp_server.go"
echo "  # Terminal 2:"
echo "  go run main.go"