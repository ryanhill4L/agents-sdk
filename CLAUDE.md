# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go SDK for building AI agents with tools, guardrails, memory, and provider integrations. The SDK follows a modular architecture with clear separation of concerns.

## Build and Development Commands

```bash
# Build the module
go build ./...

# Run tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Check module dependencies
go mod tidy

# Run tests for a specific package
go test ./pkg/agents
```

## Architecture

### Core Components

- **`pkg/agents/`** - Main agent framework
  - `Agent` - Core agent struct with tools, handoffs, guardrails
  - `Runner` - Executes agent workflows with turn management
  - Agent validation includes circular handoff detection
  - Supports both synchronous and asynchronous execution

- **`pkg/tools/`** - Tool system for agent capabilities
  - `FunctionTool` - Wraps Go functions as tools using reflection
  - Automatic schema generation from function signatures
  - Support for context-aware tool execution

- **`pkg/memory/`** - Session management and persistence
  - SQLite-based session storage for conversation history
  - Session loading and saving during agent runs

- **`pkg/providers/`** - LLM provider integrations
  - Abstraction layer for different AI providers
  - Supports completion requests with usage tracking

- **`pkg/guardrails/`** - Safety and validation system
  - Input validation before agent processing
  - Pluggable guardrail architecture

- **`pkg/tracing/`** - Observability and monitoring
  - Distributed tracing support for agent runs
  - Span tracking for debugging and performance analysis

### Key Patterns

- **Agent Handoffs**: Agents can delegate to other agents with context
- **Tool Execution**: Supports both parallel and sequential tool execution
- **Turn Management**: Configurable max turns with timeout protection
- **Structured Output**: Type-safe output schemas with validation
- **Error Handling**: Comprehensive error propagation and context

### Dependencies

- `github.com/google/uuid` - UUID generation
- `github.com/mattn/go-sqlite3` - SQLite database driver
- `golang.org/x/sync/errgroup` - Concurrent execution patterns

## Development Notes

- Missing packages (`providers`, `guardrails`, `tracing`) need implementation
- No test files currently exist - tests should be added as `*_test.go`
- Module uses Go 1.24.3
- README.md and Makefile are empty and should be populated