# Claude Code SDK for Go

A powerful Go SDK for building AI applications with Claude, following the architecture of the official Python SDK. Features a dual interface pattern for both simple queries and complex conversational interactions, with comprehensive tool support, permissions, hooks, and streaming capabilities.

[![Go Reference](https://pkg.go.dev/badge/github.com/ryanhill4L/agents-sdk.svg)](https://pkg.go.dev/github.com/ryanhill4L/agents-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

### **Dual Interface Pattern**
- **Simple Query** - Stateless, one-off queries for quick interactions
- **Interactive Client** - Stateful conversations with session management

### **Comprehensive Tool System**
- **Function Tools** - Convert any Go function into an AI-callable tool
- **Automatic Schema Generation** - Type-safe parameter validation
- **Permission System** - Granular control over tool execution
- **MCP-Compatible** - Support for Model Context Protocol patterns

### **Advanced Capabilities**
- **Streaming Support** - Real-time response streaming with channels
- **Hook System** - Event-driven architecture for intercepting operations
- **Session Management** - SQLite-based conversation persistence
- **Permission Controls** - Fine-grained access control for tools
- **Error Handling** - Comprehensive error types with retry logic

### **Built on Anthropic's Official SDK**
- Uses the official `anthropic-sdk-go` for all API interactions
- Full support for Claude 3.5 Sonnet and other models
- Automatic token counting and usage tracking
- Prompt caching support

## Installation

```bash
go get github.com/ryanhill4L/agents-sdk
```

## Quick Start

### Simple Query

```go
package main

import (
    "fmt"
    "log"

    claudecode "github.com/ryanhill4L/agents-sdk"
)

func main() {
    result, err := claudecode.Query(
        "What is the capital of France?",
        claudecode.WithAPIKey("your-api-key"),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Content)
}
```

### Interactive Client

```go
package main

import (
    "context"
    "fmt"
    "log"

    claudecode "github.com/ryanhill4L/agents-sdk"
)

func main() {
    client, err := claudecode.NewClient(
        claudecode.WithAPIKey("your-api-key"),
        claudecode.WithSystemPrompt("You are a helpful assistant."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    response, err := client.SendMessage(
        context.Background(),
        "Hello! How can you help me today?",
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.GetContent())
}
```

## Architecture

This SDK follows the architecture of the official Python Claude Code SDK, adapted for Go idioms:

```
claudecode/
├── pkg/
│   ├── client/          # Interactive conversation client
│   ├── query/           # Stateless query interface
│   ├── types/           # Core types and messages
│   ├── transport/       # Anthropic SDK integration
│   ├── tools/           # Tool system and function tools
│   ├── permissions/     # Permission management
│   ├── hooks/           # Event hook system
│   ├── memory/          # Session management
│   └── errors/          # Custom error types
└── examples/            # Usage examples
```

## Core Concepts

### Messages

The SDK uses a comprehensive message system with content blocks:

```go
// Message types
userMsg := types.NewUserMessage("Hello")
assistantMsg := types.NewAssistantMessage("Hi there!")
systemMsg := types.NewSystemMessage("You are helpful")
toolMsg := types.NewToolMessage(id, name, content, isError)

// Content blocks
textBlock := &types.TextBlock{Text: "Hello"}
toolUseBlock := &types.ToolUseBlock{Name: "calculator", Arguments: args}
toolResultBlock := &types.ToolResultBlock{Content: result}
```

### Tools

Convert any Go function into a tool:

```go
func add(a, b float64) float64 {
    return a + b
}

tool := tools.Function("add", "Add two numbers", add)
client.RegisterTool(tool)

response, _ := client.SendMessage(ctx, "What is 5 + 3?")
// Claude will automatically use the tool to calculate
```

### Permissions

Control tool execution with granular permissions:

```go
client, _ := claudecode.NewClient(
    claudecode.WithPermissionMode(types.PermissionDefault),
    claudecode.WithAllowedDirectories("./safe"),
    claudecode.WithBlockedDirectories("/etc", "/usr"),
)

// Custom permission callback
client.SetPermissionCallback(func(req types.PermissionRequest) types.PermissionResponse {
    if req.Tool == "dangerous_operation" {
        return types.PermissionResponse{
            Allowed: false,
            Reason: "Operation not permitted",
        }
    }
    return types.PermissionResponse{Allowed: true}
})
```

### Hooks

Intercept and modify behavior with hooks:

```go
loggingHook := types.NewSimpleHook(
    "logger",
    []types.HookEvent{types.HookPreMessage, types.HookPostMessage},
    func(ctx *types.HookContext) error {
        log.Printf("Event: %s", ctx.Event)
        return nil
    },
)

client.RegisterHook(loggingHook)
```

### Streaming

Real-time response streaming:

```go
client, _ := claudecode.NewClient(
    claudecode.WithStreaming(true),
)

chunks, _ := client.SendMessageStreaming(ctx, "Tell me a story")

for chunk := range chunks {
    if chunk.Content != "" {
        fmt.Print(chunk.Content)
    }
}
```

### Sessions

Persistent conversation history:

```go
client, _ := claudecode.NewClient(
    claudecode.WithSession("./sessions.db", "user-123"),
)

// Conversation history is automatically saved and restored
```

## Configuration Options

```go
// Available options
claudecode.WithAPIKey("sk-...")           // API key
claudecode.WithModel("claude-3-5-sonnet") // Model selection
claudecode.WithSystemPrompt("...")        // System instructions
claudecode.WithMaxTurns(10)               // Max conversation turns
claudecode.WithMaxTokens(4096)            // Max response tokens
claudecode.WithTemperature(0.7)           // Creativity control
claudecode.WithStreaming(true)            // Enable streaming
claudecode.WithTimeout(2 * time.Minute)   // Request timeout
claudecode.WithDebug(true)                // Debug logging

// Permission options
claudecode.WithPermissionMode(mode)       // Permission mode
claudecode.WithAllowedDirectories(dirs)   // Allowed paths
claudecode.WithBlockedDirectories(dirs)   // Blocked paths

// Session options
claudecode.WithSession(dbPath, sessionID) // Enable sessions
```

## Permission Modes

- `PermissionDefault` - Standard permissions, callbacks for sensitive operations
- `PermissionAcceptAll` - Allow all operations (use with caution)
- `PermissionAcceptEdits` - Allow read/write, prompt for delete/execute
- `PermissionBypass` - Skip all permission checks
- `PermissionRejectAll` - Reject all tool operations

## Examples

See the `examples/` directory for complete examples:

- `simple_query/` - Basic stateless queries
- `interactive_client/` - Interactive conversation client
- `tools/` - Function tools and calculations
- `streaming/` - Real-time streaming responses
- `permissions/` - Permission system demo
- `hooks/` - Event hooks and logging

## Environment Variables

```bash
export ANTHROPIC_API_KEY="your-api-key"
```

## Error Handling

The SDK provides comprehensive error types:

```go
if err != nil {
    switch {
    case errors.Is(err, errors.ErrNoAPIKey):
        // Handle missing API key
    case errors.Is(err, errors.ErrMaxTurnsExceeded):
        // Handle conversation limit
    case errors.Is(err, errors.ErrPermissionDenied):
        // Handle permission denial
    case errors.IsRetryable(err):
        // Retry the operation
    }
}
```

## Migration from Previous Version

This is a complete rewrite following the Python SDK architecture. Key changes:

1. **Dual Interface** - Use `Query()` for simple requests, `Client` for conversations
2. **No Agent Handoffs** - Removed in favor of tool-based patterns
3. **No Multi-Provider** - Focused solely on Anthropic's Claude
4. **New Permission System** - Replaces guardrails with permissions
5. **Hook System** - Replaces tracing with event hooks

## Requirements

- Go 1.19 or higher
- Anthropic API key

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built on [Anthropic's official Go SDK](https://github.com/anthropics/anthropic-sdk-go)
- Architecture inspired by [Claude Code SDK Python](https://github.com/anthropics/claude-code-sdk-python)

---

Built with ❤️ for the Go community