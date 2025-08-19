# Go Agents SDK

A powerful Go SDK for building AI agents with tools, guardrails, memory, and multi-provider support. Create sophisticated AI systems that can make decisions, use tools, collaborate through handoffs, and maintain conversation context.

[![Go Reference](https://pkg.go.dev/badge/github.com/ryanhill4L/agents-sdk.svg)](https://pkg.go.dev/github.com/ryanhill4L/agents-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/ryanhill4L/agents-sdk)](https://goreportcard.com/report/github.com/ryanhill4L/agents-sdk)

## Features

### **Multi-Provider AI Integration**
- **OpenAI** - GPT-4, GPT-4 Turbo, GPT-3.5 Turbo
- **Anthropic** - Claude 3.5 Sonnet, Claude 3 Opus, Claude 3 Haiku  
- **Google** - Gemini 2.0 Flash, Gemini 1.5 Pro
- Unified API across all providers with automatic failover

### **Powerful Tool System**
- **Function Tools** - Convert Go functions into AI-callable tools with automatic schema generation
- **Type Safety** - Automatic parameter validation and type conversion
- **Parallel Execution** - Tools can run concurrently for improved performance
- **Error Handling** - Comprehensive error propagation and context

### **Agent Handoffs & Orchestration**
- **Smart Routing** - Agents can delegate tasks to specialized sub-agents
- **Context Preservation** - Full conversation context maintained across handoffs
- **Circular Detection** - Prevents infinite handoff loops with validation
- **Execution Tracking** - Complete audit trail of agent interactions

### **Guardrails & Safety**
- **Input Validation** - Screen requests before processing
- **Output Filtering** - Validate responses before returning
- **Custom Guardrails** - Implement domain-specific safety checks
- **Privacy Protection** - Built-in data protection patterns

### **Memory & Sessions**
- **SQLite Backend** - Persistent conversation storage
- **Session Management** - Load and save conversation history
- **Context Windows** - Intelligent context management for long conversations
- **Multi-Session** - Support for concurrent user sessions

### **Observability & Monitoring**
- **Distributed Tracing** - Full request/response tracing
- **Performance Metrics** - Token usage, latency, and success rates
- **Debug Logging** - Detailed execution logs for troubleshooting
- **Console Tracer** - Built-in development debugging

## Quick Start

### Installation

```bash
go get github.com/ryanhill4L/agents-sdk
```

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ryanhill4L/agents-sdk/pkg/agents"
    "github.com/ryanhill4L/agents-sdk/pkg/providers"
    "github.com/ryanhill4L/agents-sdk/pkg/tools"
    "github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// Define a simple tool
func add(a, b int) int {
    return a + b
}

func main() {
    // Create a tool from your function
    addTool, err := tools.NewFunctionTool("add", "Adds two numbers", add)
    if err != nil {
        log.Fatal(err)
    }

    // Create an agent
    agent := agents.NewAgent("Math Assistant",
        agents.WithInstructions("You are a helpful math assistant."),
        agents.WithModel("gpt-4"),
        agents.WithTools(addTool),
    )

    // Create a provider (OpenAI, Anthropic, or Gemini)
    provider, err := providers.NewOpenAIProviderFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Create a runner
    runner := agents.NewRunner(
        agents.WithProvider(provider),
        agents.WithTracer(tracing.NewConsoleTracer()),
    )

    // Run the agent
    ctx := context.Background()
    result, err := runner.Run(ctx, agent, "What is 15 + 27?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Response: %s\n", result.FinalOutput)
    fmt.Printf("Tokens: %d, Duration: %v\n", 
        result.Metrics.TotalTokens, result.Metrics.Duration)
}
```

## Examples

### Event Scheduling Agent
A complete example showing agent handoffs, database tools, and guardrails.

```bash
cd examples/event-scheduler
export OPENAI_API_KEY="your-key-here"
go run main.go
```

**Features:**
- Multi-agent orchestration with triage routing
- Database integration with SQLite
- Conflict detection and scheduling optimization
- Privacy guardrails for sensitive data

### Basic Tools Demo
Simple demonstration of function tools and multi-provider support.

```bash
cd examples/basic
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export GEMINI_API_KEY="your-gemini-key"
go run main.go
```

### JSON Structure Example
Shows structured output with type-safe JSON parsing.

```bash
cd examples/json
go run main.go
```

## Architecture

### Core Components

```
agents-sdk/
├── pkg/
│   ├── agents/          # Core agent framework
│   │   ├── agent.go     # Agent definition and configuration
│   │   ├── runner.go    # Execution engine with turn management
│   │   └── types.go     # Type definitions and interfaces
│   ├── tools/           # Tool system for agent capabilities
│   │   ├── function_tool.go  # Go function to tool conversion
│   │   └── tools.go     # Tool interfaces and utilities
│   ├── providers/       # AI model provider integrations
│   │   ├── openai_provider.go    # OpenAI GPT models
│   │   ├── anthropic_provider.go # Anthropic Claude models
│   │   └── gemini_provider.go    # Google Gemini models
│   ├── memory/          # Session management and persistence
│   │   ├── session.go   # Session interface
│   │   └── sqlite_session.go # SQLite implementation
│   ├── guardrails/      # Safety and validation system
│   │   └── guardrail.go # Guardrail interfaces
│   └── tracing/         # Observability and monitoring
│       └── tracer.go    # Tracing interfaces and console tracer
└── examples/            # Complete working examples
    ├── event-scheduler/ # Multi-agent scheduling system
    ├── basic/          # Simple tool demonstration
    └── json/           # Structured output example
```

### Agent Lifecycle

1. **Initialization** - Agent created with tools, instructions, and configuration
2. **Validation** - Check for circular handoffs and required dependencies
3. **Execution** - Runner manages conversation turns and tool calls
4. **Tool Calling** - Automatic function execution with type conversion
5. **Handoffs** - Optional delegation to specialized agents
6. **Response** - Final output with metrics and tracing data

## Configuration

### Environment Variables

```bash
# AI Provider API Keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."  
export GEMINI_API_KEY="..."

# Optional Configuration
export AGENTS_DEBUG=true
export AGENTS_TRACE_LEVEL=debug
export AGENTS_MAX_TOKENS=4096
```

### Provider Configuration

```go
// OpenAI
provider, err := providers.NewOpenAIProviderWithKey("sk-...")

// Anthropic
provider, err := providers.NewAnthropicProviderWithKey("sk-ant-...")

// Gemini
provider, err := providers.NewGeminiProviderWithKey("gemini-key")

// From environment
provider, err := providers.NewOpenAIProviderFromEnv()
```

### Agent Configuration

```go
agent := agents.NewAgent("Agent Name",
    agents.WithInstructions("Your role and instructions"),
    agents.WithModel("gpt-4"),
    agents.WithTools(tool1, tool2),
    agents.WithHandoffs(subAgent1, subAgent2),
    agents.WithGuardrails(guardrail),
    agents.WithTemperature(0.7),
)
```

### Runner Configuration

```go
runner := agents.NewRunner(
    agents.WithProvider(provider),
    agents.WithTracer(tracing.NewConsoleTracer()),
    agents.WithMaxTurns(10),
    agents.WithParallelTools(true),
)
```

## Advanced Features

### Custom Tools

```go
// Database query tool
func queryDB(query string) ([]map[string]any, error) {
    // Your database logic
    return results, nil
}

tool, err := tools.NewFunctionTool(
    "query_database", 
    "Execute SQL queries", 
    queryDB,
)
```

### Agent Handoffs

```go
// Specialized agents
dataAgent := agents.NewAgent("Data Analyst", ...)
reportAgent := agents.NewAgent("Report Generator", ...)

// Orchestrator with handoffs
orchestrator := agents.NewAgent("Coordinator",
    agents.WithHandoffs(dataAgent, reportAgent),
    agents.WithInstructions("Route tasks to specialists..."),
)
```

### Memory Integration

```go
// Session-aware runner
session, err := memory.NewSQLiteSession("./sessions.db", "user123")
runner := agents.NewRunner(
    agents.WithProvider(provider),
    agents.WithSession(session),
)

// Conversation history is automatically preserved
```

### Custom Guardrails

```go
type MyGuardrail struct{}

func (g *MyGuardrail) Name() string { return "my_check" }
func (g *MyGuardrail) Description() string { return "Custom validation" }
func (g *MyGuardrail) Validate(content string) error {
    if containsSensitiveData(content) {
        return fmt.Errorf("sensitive data detected")
    }
    return nil
}

agent := agents.NewAgent("Secure Agent",
    agents.WithGuardrails(&MyGuardrail{}),
)
```

## API Reference

### Core Types

```go
type Agent struct {
    // Agent configuration and state
}

type RunResult struct {
    FinalOutput string
    Metrics     RunMetrics
    TraceID     string
}

type RunMetrics struct {
    TotalTurns   int
    ToolCalls    int
    Handoffs     int
    TotalTokens  int
    Duration     time.Duration
}
```

### Key Interfaces

```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args map[string]any) (any, error)
}

type Guardrail interface {
    Name() string
    Description() string
    Validate(content string) error
}
```

## Performance & Scalability

- **Concurrent Tool Execution** - Multiple tools can run in parallel
- **Efficient Context Management** - Smart context window handling
- **Connection Pooling** - Reused HTTP connections to providers
- **Memory Optimization** - Efficient session storage and retrieval
- **Timeout Handling** - Configurable timeouts for all operations

## Security & Privacy

- **API Key Management** - Secure credential handling
- **Input Sanitization** - Automatic validation of user inputs
- **Output Filtering** - Guardrails for response validation
- **Data Isolation** - Session-based data separation
- **Audit Logging** - Complete tracing of all operations

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
git clone https://github.com/ryanhill4L/agents-sdk.git
cd agents-sdk
go mod download
go test ./...
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./pkg/agents
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [pkg.go.dev](https://pkg.go.dev/github.com/ryanhill4L/agents-sdk)
- **Quickstart**: See [QUICKSTART.md](QUICKSTART.md)
- **Discussions**: [GitHub Discussions](https://github.com/ryanhill4L/agents-sdk/discussions)
- **Bug Reports**: [GitHub Issues](https://github.com/ryanhill4L/agents-sdk/issues)

## Roadmap

- [ ] **Stream Processing** - Real-time streaming responses
- [ ] **Plugin System** - Dynamic tool loading
- [ ] **Workflow Engine** - Complex multi-step processes
- [ ] **Vector Memory** - Semantic memory with embeddings
- [ ] **Web Interface** - Browser-based agent management
- [ ] **Kubernetes Operator** - Cloud-native deployment

---

Built with ❤️ by [Ryan Hill](https://github.com/ryanhill4L)