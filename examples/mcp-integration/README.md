# MCP Integration Example

This example demonstrates how to integrate **Model Context Protocol (MCP)** support into the Go Agents SDK. MCP enables agents to interact with external tools and services through a standardized protocol, greatly expanding their capabilities.

## What is MCP?

Model Context Protocol (MCP) is a standardized protocol that allows AI agents to communicate with external tools, data sources, and services. It provides:

- **Standardized Communication**: Consistent API for tool discovery and execution
- **Dynamic Tool Loading**: Agents can discover and use tools at runtime
- **Extensibility**: Easy to add new capabilities without modifying agent code
- **Security**: Controlled access to external resources

## Features Demonstrated

This example showcases:

### 🔧 **MCP Tool Integration**
- Dynamic tool discovery from MCP servers
- Automatic schema generation for external tools
- Seamless integration with existing agent workflows

### 📁 **File System Operations**
- List directory contents
- Read file contents
- Write files with automatic directory creation

### 🌤️ **External API Integration**
- Weather information retrieval
- Current time/date services

### 🔀 **Mixed Tool Usage**
- Combining MCP tools with local function tools
- Parallel tool execution across different sources

### 🛡️ **Security & Error Handling**
- Input validation and sanitization
- Proper error propagation
- Timeout handling for external requests

## Project Structure

```
examples/mcp-integration/
├── README.md           # This documentation
├── go.mod             # Go module definition
├── main.go            # Agent example using MCP tools
├── mcp_server.go      # Example MCP server implementation
└── pkg/tools/         # (in parent directory)
    └── mcp_tool.go    # MCP tool implementation
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- OpenAI API key (set in environment)

### 1. Set Environment Variables

```bash
export OPENAI_API_KEY="your-openai-api-key-here"
```

### 2. Start the MCP Server

In one terminal, start the example MCP server:

```bash
cd examples/mcp-integration
go run mcp_server.go
```

You should see:
```
🔧 MCP Server starting...
Available tools:
  - list_files: List files in a directory
  - read_file: Read contents of a text file
  - write_file: Write content to a text file
  - get_weather: Get current weather information for a city
  - get_current_time: Get the current date and time

Server running on http://localhost:8080
Use Ctrl+C to stop the server
```

### 3. Run the Agent Example

In another terminal, run the agent example:

```bash
cd examples/mcp-integration
go run main.go
```

## Example Output

```
🤖 Agents SDK - MCP Integration Example
=====================================

🔍 Checking MCP server availability...
✅ MCP server is running with 5 tools available
Available tools: [list_files read_file write_file get_weather get_current_time]

🔧 Creating MCP tools...
✅ Created 5 MCP tools successfully

🚀 Running MCP integration examples...

📋 Example 1: Weather Query
💬 Query: What's the weather like in San Francisco?
📝 Expected: This will use the MCP get_weather tool to fetch weather data from the external server.
⏳ Processing...

✅ Response: I'll check the current weather in San Francisco for you.

The current weather in San Francisco is:
- Temperature: 22°C (72°F)
- Condition: Partly cloudy
- Humidity: 65%
- Wind Speed: 10 km/h
- Last updated: 2024-01-15T10:30:45Z

The weather looks quite pleasant with partly cloudy skies and comfortable temperatures!

📊 Metrics: 2 turns, 1 tool calls, 856 tokens, 2.1s
```

## MCP Tool Implementation

### Core Components

#### 1. MCPTool Structure

```go
type MCPTool struct {
    name        string
    description string
    schema      ParameterSchema
    serverURL   string
    client      *http.Client
    toolName    string
}
```

#### 2. MCPClient for Discovery

```go
type MCPClient struct {
    serverURL string
    client    *http.Client
}

// Create client and discover tools
client := tools.NewMCPClient("http://localhost:8080")
toolNames, err := client.ListTools()
weatherTool, err := client.GetTool("get_weather")
```

#### 3. Protocol Implementation

The MCP implementation supports these standard methods:

- **`tools/list`**: Discover available tools
- **`tools/get`**: Get tool schema and metadata  
- **`tools/call`**: Execute tool with parameters

### Tool Creation Patterns

#### Option 1: Direct Tool Creation

```go
config := tools.MCPToolConfig{
    Name:        "weather_checker",
    Description: "Check weather for any city",
    ServerURL:   "http://localhost:8080",
    ToolName:    "get_weather",
    Schema: tools.ParameterSchema{
        Type: "object",
        Properties: map[string]tools.PropertySchema{
            "city": {Type: "string", Description: "City name"},
        },
        Required: []string{"city"},
    },
}

tool, err := tools.NewMCPTool(config)
```

#### Option 2: Dynamic Discovery

```go
client := tools.NewMCPClient("http://localhost:8080")
tool, err := client.GetTool("get_weather")
```

#### Option 3: Bulk Tool Loading

```go
client := tools.NewMCPClient("http://localhost:8080")
allTools, err := client.GetAllTools()
```

## Available MCP Tools

### File System Tools

#### `list_files`
Lists files and directories in a specified path.

**Parameters:**
- `directory` (string): Directory path to list

**Example:**
```json
{
  "directory": ".",
  "files": [
    {"name": "main.go", "isDir": false},
    {"name": "examples", "isDir": true}
  ]
}
```

#### `read_file`
Reads and returns the contents of a text file.

**Parameters:**
- `filepath` (string): Path to file to read

**Example:**
```json
{
  "filepath": "example.txt",
  "content": "File contents here...",
  "size": 1024
}
```

#### `write_file`
Writes content to a file, creating directories as needed.

**Parameters:**
- `filepath` (string): Path where to write file
- `content` (string): Content to write

**Example:**
```json
{
  "filepath": "output.txt",
  "size": 256,
  "status": "success"
}
```

### Information Tools

#### `get_weather`
Retrieves weather information for a specified city.

**Parameters:**
- `city` (string): Name of the city
- `units` (string, optional): "celsius" or "fahrenheit"

**Example:**
```json
{
  "city": "San Francisco",
  "temperature": 22,
  "units": "celsius",
  "condition": "Partly cloudy",
  "humidity": 65,
  "windSpeed": "10 km/h",
  "timestamp": "2024-01-15T10:30:45Z"
}
```

#### `get_current_time`
Gets current date and time information.

**Parameters:**
- `timezone` (string, optional): Timezone name (defaults to UTC)

**Example:**
```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "timezone": "UTC", 
  "unix": 1705317045,
  "formatted": "2024-01-15 10:30:45"
}
```

## Agent Integration

### Creating an Agent with MCP Tools

```go
// Discover and create MCP tools
mcpClient := tools.NewMCPClient("http://localhost:8080")
weatherTool, _ := mcpClient.GetTool("get_weather")
filesTool, _ := mcpClient.GetTool("list_files")

// Create local tools
calcTool, _ := tools.NewFunctionTool("calc", "Calculator", calculate)

// Create agent with mixed tools
agent := agents.NewAgent("MCP Assistant",
    agents.WithInstructions("You can use both MCP and local tools..."),
    agents.WithModel("gpt-4"),
    agents.WithTools(weatherTool, filesTool, calcTool),
)
```

### Tool Usage in Conversations

The agent automatically chooses appropriate tools based on user queries:

```
User: "What's the weather in Tokyo and list files in /tmp?"

Agent: I'll help you with both requests.

[Uses get_weather tool for Tokyo]
[Uses list_files tool for /tmp]

Agent: The weather in Tokyo is 18°C and partly cloudy. 
       The /tmp directory contains 12 files including...
```

## Error Handling

### Common Error Scenarios

#### 1. MCP Server Unavailable
```go
toolNames, err := mcpClient.ListTools()
if err != nil {
    log.Printf("MCP server not available: %v", err)
    // Fallback to local tools only
}
```

#### 2. Tool Execution Errors
```go
result, err := tool.Execute(ctx, args)
if err != nil {
    // Handle tool-specific errors
    log.Printf("Tool execution failed: %v", err)
}
```

#### 3. Invalid Parameters
The SDK automatically validates parameters against the tool schema before execution.

## Security Considerations

### Input Validation
- All file paths are checked for directory traversal attempts
- Parameters are validated against tool schemas
- Timeout handling prevents hanging requests

### Network Security
- HTTPS support for production MCP servers
- Configurable timeouts and retry policies
- Request/response logging for audit trails

### Access Control
```go
// Example: Restrict file operations to specific directories
if !strings.HasPrefix(filepath, "/allowed/path/") {
    return nil, fmt.Errorf("access denied")
}
```

## Advanced Usage

### Custom MCP Servers

You can create your own MCP servers to provide domain-specific tools:

```go
type CustomMCPServer struct {
    tools map[string]*Tool
}

func (s *CustomMCPServer) RegisterTool(name string, tool *Tool) {
    s.tools[name] = tool
}

// Implement database tools, API integrations, etc.
```

### Tool Composition

Combine multiple MCP servers for different capabilities:

```go
// Database operations
dbClient := tools.NewMCPClient("http://db-server:8080")
dbTools, _ := dbClient.GetAllTools()

// File operations
fileClient := tools.NewMCPClient("http://file-server:8080") 
fileTools, _ := fileClient.GetAllTools()

// Combine all tools
allTools := append(dbTools, fileTools...)
agent := agents.NewAgent("Multi-MCP Agent", 
    agents.WithTools(allTools...))
```

### Production Deployment

For production use:

1. **Use HTTPS**: Configure TLS for MCP servers
2. **Authentication**: Implement API key or OAuth validation
3. **Rate Limiting**: Prevent abuse of external tools
4. **Monitoring**: Track tool usage and performance
5. **Caching**: Cache tool results where appropriate

## Troubleshooting

### Common Issues

#### MCP Server Connection Failed
```
❌ MCP server is not running or not accessible: dial tcp [::1]:8080: connect: connection refused
```
**Solution**: Ensure the MCP server is running on the correct port.

#### Tool Not Found
```
❌ Tool not found: invalid_tool
```
**Solution**: Check available tools with `ListTools()` and verify tool names.

#### Permission Denied
```
❌ failed to read file: permission denied
```
**Solution**: Check file permissions and path accessibility.

### Debug Mode

Enable debug logging to see detailed MCP communication:

```go
runner := agents.NewRunner(
    agents.WithProvider(provider),
    agents.WithTracer(tracing.NewConsoleTracer()),
    // Add debug configuration
)
```

## Next Steps

### Extending the Example

1. **Add Authentication**: Implement API key validation
2. **Add More Tools**: Create database, email, or API tools  
3. **Error Recovery**: Add retry logic and fallback strategies
4. **Streaming**: Implement streaming responses for long operations
5. **Caching**: Add response caching for expensive operations

### Integration Patterns

- **Microservices**: Each service provides its own MCP server
- **Plugin Architecture**: Dynamic tool loading at runtime
- **Multi-Tenant**: Per-user tool access controls
- **Event-Driven**: Tools that trigger on external events

## Resources

- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [Go Agents SDK Documentation](../../README.md)
- [OpenAI Function Calling Guide](https://platform.openai.com/docs/guides/function-calling)

## Contributing

To add more MCP tools or improve the implementation:

1. Fork the repository
2. Add your tools to the MCP server
3. Update the agent example
4. Add tests and documentation
5. Submit a pull request

---

**Built with ❤️ using the Go Agents SDK and Model Context Protocol**