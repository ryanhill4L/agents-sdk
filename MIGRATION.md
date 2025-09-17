# Migration Guide

## Migrating from v1 to v2

This is a complete rewrite of the SDK following the architecture of the official Python Claude Code SDK. The new version focuses exclusively on Anthropic's Claude API and introduces a dual interface pattern.

## Major Changes

### 1. Package Structure

**Old:**
```go
import (
    "github.com/ryanhill4L/agents-sdk/pkg/agents"
    "github.com/ryanhill4L/agents-sdk/pkg/providers"
    "github.com/ryanhill4L/agents-sdk/pkg/tools"
)
```

**New:**
```go
import (
    claudecode "github.com/ryanhill4L/agents-sdk"
    "github.com/ryanhill4L/agents-sdk/pkg/types"
    "github.com/ryanhill4L/agents-sdk/pkg/tools"
)
```

### 2. Simple Queries (New Feature)

For simple, stateless queries, use the new `Query` function:

```go
// New simple interface
result, err := claudecode.Query(
    "What is 2+2?",
    claudecode.WithAPIKey("sk-..."),
)
fmt.Println(result.Content)
```

### 3. Client Interface (Replaces Agent/Runner)

**Old:**
```go
agent := agents.NewAgent("Assistant",
    agents.WithInstructions("You are helpful"),
    agents.WithModel("gpt-4"),
)
runner := agents.NewRunner(
    agents.WithProvider(provider),
)
result, err := runner.Run(ctx, agent, "Hello")
```

**New:**
```go
client, err := claudecode.NewClient(
    claudecode.WithAPIKey("sk-..."),
    claudecode.WithSystemPrompt("You are helpful"),
    claudecode.WithModel("claude-3-5-sonnet-20241022"),
)
response, err := client.SendMessage(ctx, "Hello")
```

### 4. Providers → Transport

The multi-provider system has been replaced with a focused Anthropic transport:

**Old:**
```go
// Multiple provider options
provider, _ := providers.NewOpenAIProviderFromEnv()
provider, _ := providers.NewAnthropicProviderFromEnv()
provider, _ := providers.NewGeminiProviderFromEnv()
```

**New:**
```go
// Single Anthropic-focused client
client, _ := claudecode.NewClient(
    claudecode.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
)
```

### 5. Tools

Tool creation remains similar but with improved type safety:

**Old:**
```go
tool, _ := tools.NewFunctionTool("add", "Adds numbers", add)
agent.Tools = append(agent.Tools, tool)
```

**New:**
```go
tool := tools.Function("add", "Adds numbers", add)
client.RegisterTool(tool)
```

### 6. Agent Handoffs → Tools

Agent handoffs have been removed. Use tools for delegation:

**Old:**
```go
subAgent := agents.NewAgent("Specialist", ...)
mainAgent := agents.NewAgent("Main",
    agents.WithHandoffs(subAgent),
)
```

**New:**
```go
// Create a tool that encapsulates specialist functionality
specialistTool := tools.Function("specialist", "...", specialistFunc)
client.RegisterTool(specialistTool)
```

### 7. Guardrails → Permissions

**Old:**
```go
type MyGuardrail struct{}
func (g *MyGuardrail) Validate(content string) error { ... }

agent := agents.NewAgent("Agent",
    agents.WithGuardrails(&MyGuardrail{}),
)
```

**New:**
```go
client.SetPermissionMode(types.PermissionDefault)
client.SetPermissionCallback(func(req types.PermissionRequest) types.PermissionResponse {
    // Your validation logic
    return types.PermissionResponse{
        Allowed: true,
        Reason: "...",
    }
})
```

### 8. Tracing → Hooks

**Old:**
```go
runner := agents.NewRunner(
    agents.WithTracer(tracing.NewConsoleTracer()),
)
```

**New:**
```go
hook := types.NewSimpleHook("logger",
    []types.HookEvent{types.HookPreMessage, types.HookPostMessage},
    func(ctx *types.HookContext) error {
        log.Printf("Event: %s", ctx.Event)
        return nil
    },
)
client.RegisterHook(hook)
```

### 9. Sessions

Session management is now built into the client:

**Old:**
```go
session, _ := memory.NewSQLiteSession("user123", "sessions.db")
// Manual session management
```

**New:**
```go
client, _ := claudecode.NewClient(
    claudecode.WithSession("./sessions.db", "user123"),
)
// Automatic session management
```

### 10. Streaming (New Feature)

Native streaming support:

```go
client, _ := claudecode.NewClient(
    claudecode.WithStreaming(true),
)

chunks, _ := client.SendMessageStreaming(ctx, "Tell me a story")
for chunk := range chunks {
    fmt.Print(chunk.Content)
}
```

## Step-by-Step Migration

### Step 1: Update Imports

Replace all old package imports with the new structure:

```go
// Replace these
import (
    "github.com/ryanhill4L/agents-sdk/pkg/agents"
    "github.com/ryanhill4L/agents-sdk/pkg/providers"
    "github.com/ryanhill4L/agents-sdk/pkg/guardrails"
    "github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// With this
import (
    claudecode "github.com/ryanhill4L/agents-sdk"
    "github.com/ryanhill4L/agents-sdk/pkg/types"
)
```

### Step 2: Replace Agent/Runner with Client

Convert your agent and runner setup to use the new client:

```go
// Old
agent := agents.NewAgent(...)
runner := agents.NewRunner(...)
result, _ := runner.Run(ctx, agent, prompt)

// New
client, _ := claudecode.NewClient(...)
response, _ := client.SendMessage(ctx, prompt)
```

### Step 3: Update Tool Registration

Change how tools are registered:

```go
// Old
agent.Tools = []tools.Tool{tool1, tool2}

// New
client.RegisterTool(tool1)
client.RegisterTool(tool2)
```

### Step 4: Convert Guardrails to Permissions

Replace guardrail validation with permission callbacks:

```go
// Old guardrail validation
type ContentGuardrail struct{}
func (g *ContentGuardrail) Validate(content string) error {
    if hasSensitiveData(content) {
        return fmt.Errorf("sensitive data detected")
    }
    return nil
}

// New permission callback
client.SetPermissionCallback(func(req types.PermissionRequest) types.PermissionResponse {
    if hasSensitiveData(req.Arguments) {
        return types.PermissionResponse{
            Allowed: false,
            Reason: "sensitive data detected",
        }
    }
    return types.PermissionResponse{Allowed: true}
})
```

### Step 5: Replace Tracing with Hooks

Convert tracer usage to hook registration:

```go
// Old
runner := agents.NewRunner(
    agents.WithTracer(customTracer),
)

// New
hook := types.NewSimpleHook("custom", events, hookFunc)
client.RegisterHook(hook)
```

## Feature Comparison

| Feature | v1 | v2 |
|---------|----|----|
| Simple Queries | ❌ | ✅ |
| Stateful Client | Via Agent/Runner | ✅ Native Client |
| Multi-Provider | ✅ | ❌ Anthropic Only |
| Agent Handoffs | ✅ | ❌ Use Tools |
| Guardrails | ✅ | → Permissions |
| Tracing | ✅ | → Hooks |
| Streaming | ❌ | ✅ |
| Session Management | Manual | ✅ Automatic |
| Tool System | ✅ | ✅ Enhanced |
| Type Safety | Partial | ✅ Full |

## Common Patterns

### Pattern 1: Simple Query

```go
// Stateless, one-off query
result, _ := claudecode.Query("What is the weather?")
```

### Pattern 2: Conversation with Tools

```go
client, _ := claudecode.NewClient(...)
client.RegisterTool(weatherTool)
client.RegisterTool(calculatorTool)

response, _ := client.SendMessage(ctx, "What's the weather and calculate the humidity percentage")
```

### Pattern 3: Secured Operations

```go
client, _ := claudecode.NewClient(
    claudecode.WithPermissionMode(types.PermissionDefault),
    claudecode.WithAllowedDirectories("./safe"),
)

client.SetPermissionCallback(func(req types.PermissionRequest) types.PermissionResponse {
    // Custom security logic
    return types.PermissionResponse{Allowed: userApproved}
})
```

## Breaking Changes Summary

1. **No multi-provider support** - Only Anthropic Claude is supported
2. **No agent handoffs** - Use tools for delegation
3. **Guardrails replaced with permissions** - New permission system
4. **Tracing replaced with hooks** - Event-driven architecture
5. **New package structure** - Simplified imports
6. **Required API key** - No default provider detection

## Need Help?

- Check the [examples/](examples/) directory for working code
- Review the [README.md](README.md) for detailed documentation
- File issues on GitHub for migration problems