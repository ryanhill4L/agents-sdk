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
  - `SimpleTool` (`NewTool`) - Tools with explicit, named parameter schemas
  - `Registry` - Name-indexed tool collection used by the filesystem loader
  - Support for context-aware tool execution

- **`pkg/loader/`** - Filesystem-first agent loading
  - Builds an `Agent` from a directory (`agent.yaml`, `instructions.md`, `skills/`, `subagents/`)
  - Resolves tools declared by name in `agent.yaml` against a `tools.Registry`
  - Loads subagents recursively as handoffs

- **`pkg/skills/`** - On-demand knowledge (Eve-style skills)
  - Markdown files with YAML front-matter (`name`, `description`)
  - Catalog is appended to the system prompt; bodies are pulled via the `load_skill` builtin tool
  - Remote skills (`RemoteSource`): declared in `agent.yaml`, fetched at load time, pinned by commit SHA, `sha256`-verified, host-allowlisted, and cached (never model-driven runtime fetches)

- **`pkg/mcp/`** - Model Context Protocol integration
  - Connects to MCP servers over stdio or streamable-HTTP using `modelcontextprotocol/go-sdk`
  - Adapts each server tool into the SDK's `tools.Tool` interface (namespaced as `<server>_<tool>`)
  - `Manager` aggregates multiple servers; connections close via `Agent.Close()`

- **`pkg/channels/`** - Integration adapters
  - `Channel` interface; built-in `HTTPChannel` powers `eve dev` (POST /chat)

- **`pkg/schedules/`** - Cron-based autonomous triggers
  - Dependency-free 5-field cron parser/matcher and a minute-tick scheduler

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

### The `cmd/eve` CLI

The `eve` binary is a thin shell over the library:

- `eve init <dir>` - scaffold a new agent directory
- `eve validate <dir>` - load an agent and report what was found (no API key needed)
- `eve run <dir> [input]` - run a single turn
- `eve dev <dir> [addr]` - serve the agent over HTTP via the HTTP channel
- `eve schedules <dir>` - run the agent's cron schedules

Because Go tools are compiled, the CLI resolves `agent.yaml` tool names against a
small builtin registry (`current_time`, `add`, `http_get`). Custom Go tools are
wired by calling `loader.Load(dir, registry)` from your own program.

### Key Patterns

- **Filesystem-first**: agents are directories; the loader translates files into `agents.NewAgent` option calls
- **Skills on demand**: catalog in the prompt + `load_skill` builtin tool, rather than inlining all knowledge
- **Agent Handoffs**: subagent directories become handoff targets, exposed to the model as `handoff_<name>` tools and intercepted by the Runner
- **Handoff context modes**: `shared` (transfer control, default), `fresh` (delegate with task only, returns), `forked` (delegate with a copy of history, returns); set per subagent in `agent.yaml` or via `agents.WithHandoff`
- **Tool Execution**: supports both parallel and sequential tool execution
- **Turn Management**: configurable max turns with timeout protection
- **Error Handling**: comprehensive error propagation and context

### Dependencies

- `github.com/google/uuid` - UUID generation
- `github.com/mattn/go-sqlite3` - SQLite database driver
- `golang.org/x/sync/errgroup` - Concurrent execution patterns
- `gopkg.in/yaml.v3` - YAML parsing for `agent.yaml`, skills front-matter, schedules
- `github.com/modelcontextprotocol/go-sdk` - MCP client (stdio + streamable-HTTP)
- Provider SDKs: `anthropic-sdk-go`, `openai-go`, `ollama`, `google.golang.org/genai`

## Development Notes

- Provider, guardrail, and tracing packages are implemented
- Tests live alongside code as `*_test.go` (loader, skills, schedules, tools, agents)
- Module requires Go 1.25 (raised by the MCP SDK dependency)
- Build the CLI with `make build-cli`; validate the example with `make run-fs-example`