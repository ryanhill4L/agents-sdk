package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MCPTool represents a tool that communicates with an MCP server
type MCPTool struct {
	name        string
	description string
	schema      ParameterSchema
	serverURL   string
	client      *http.Client
	toolName    string // The actual tool name on the MCP server
}

// MCPRequest represents a request to an MCP server
type MCPRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// MCPResponse represents a response from an MCP server
type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error from an MCP server
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPToolConfig contains configuration for creating an MCP tool
type MCPToolConfig struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	ServerURL   string          `json:"serverURL"`
	ToolName    string          `json:"toolName"`
	Schema      ParameterSchema `json:"schema"`
	Timeout     time.Duration   `json:"timeout,omitempty"`
}

// NewMCPTool creates a new MCP tool
func NewMCPTool(config MCPToolConfig) (*MCPTool, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("tool name cannot be empty")
	}
	if config.ServerURL == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}
	if config.ToolName == "" {
		return nil, fmt.Errorf("tool name on server cannot be empty")
	}

	// Validate server URL
	if _, err := url.Parse(config.ServerURL); err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	tool := &MCPTool{
		name:        config.Name,
		description: config.Description,
		schema:      config.Schema,
		serverURL:   config.ServerURL,
		client:      client,
		toolName:    config.ToolName,
	}

	return tool, nil
}

// NewMCPToolFromServer creates an MCP tool by querying the server for tool information
func NewMCPToolFromServer(serverURL, toolName string) (*MCPTool, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Query the server for tool information
	toolInfo, err := queryMCPServerForTool(client, serverURL, toolName)
	if err != nil {
		return nil, fmt.Errorf("failed to query MCP server for tool info: %w", err)
	}

	config := MCPToolConfig{
		Name:        toolName,
		Description: toolInfo.Description,
		ServerURL:   serverURL,
		ToolName:    toolName,
		Schema:      toolInfo.Schema,
	}

	return NewMCPTool(config)
}

// MCPToolInfo represents information about a tool from an MCP server
type MCPToolInfo struct {
	Description string          `json:"description"`
	Schema      ParameterSchema `json:"schema"`
}

// queryMCPServerForTool queries an MCP server for information about a specific tool
func queryMCPServerForTool(client *http.Client, serverURL, toolName string) (*MCPToolInfo, error) {
	// Create request to get tool info
	request := MCPRequest{
		Method: "tools/get",
		Params: map[string]interface{}{
			"name": toolName,
		},
	}

	resp, err := makeMCPRequest(client, serverURL, request)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP server error: %s", resp.Error.Message)
	}

	// Parse the tool info from the response
	toolInfoBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool info: %w", err)
	}

	var toolInfo MCPToolInfo
	if err := json.Unmarshal(toolInfoBytes, &toolInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool info: %w", err)
	}

	return &toolInfo, nil
}

// makeMCPRequest makes an HTTP request to an MCP server
func makeMCPRequest(client *http.Client, serverURL string, request MCPRequest) (*MCPResponse, error) {
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Post(serverURL, "application/json", strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d", resp.StatusCode)
	}

	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &mcpResp, nil
}

// Name returns the tool name
func (m *MCPTool) Name() string {
	return m.name
}

// Description returns the tool description
func (m *MCPTool) Description() string {
	return m.description
}

// Schema returns the parameter schema
func (m *MCPTool) Schema() ParameterSchema {
	return m.schema
}

// Execute runs the tool by calling the MCP server
func (m *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Create MCP request
	request := MCPRequest{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name":      m.toolName,
			"arguments": args,
		},
	}

	// Make request with context timeout
	client := &http.Client{
		Timeout: m.client.Timeout,
	}

	resp, err := makeMCPRequest(client, m.serverURL, request)
	if err != nil {
		return nil, fmt.Errorf("MCP request failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP tool execution failed: %s", resp.Error.Message)
	}

	return resp.Result, nil
}

// Validate checks if the tool configuration is valid
func (m *MCPTool) Validate() error {
	if m.name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if m.serverURL == "" {
		return fmt.Errorf("server URL cannot be empty")
	}
	if m.toolName == "" {
		return fmt.Errorf("tool name on server cannot be empty")
	}
	if m.client == nil {
		return fmt.Errorf("HTTP client cannot be nil")
	}
	return nil
}

// MCPClient provides a convenient way to create multiple MCP tools from a server
type MCPClient struct {
	serverURL string
	client    *http.Client
}

// NewMCPClient creates a new MCP client
func NewMCPClient(serverURL string) *MCPClient {
	return &MCPClient{
		serverURL: serverURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListTools lists all available tools on the MCP server
func (c *MCPClient) ListTools() ([]string, error) {
	request := MCPRequest{
		Method: "tools/list",
		Params: map[string]interface{}{},
	}

	resp, err := makeMCPRequest(c.client, c.serverURL, request)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP server error: %s", resp.Error.Message)
	}

	// Parse the tools list from the response
	toolsBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools list: %w", err)
	}

	var tools struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(toolsBytes, &tools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools list: %w", err)
	}

	var toolNames []string
	for _, tool := range tools.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	return toolNames, nil
}

// GetTool creates an MCP tool for the specified tool name
func (c *MCPClient) GetTool(toolName string) (*MCPTool, error) {
	return NewMCPToolFromServer(c.serverURL, toolName)
}

// GetAllTools creates MCP tools for all available tools on the server
func (c *MCPClient) GetAllTools() ([]*MCPTool, error) {
	toolNames, err := c.ListTools()
	if err != nil {
		return nil, err
	}

	var tools []*MCPTool
	for _, toolName := range toolNames {
		tool, err := c.GetTool(toolName)
		if err != nil {
			return nil, fmt.Errorf("failed to create tool %s: %w", toolName, err)
		}
		tools = append(tools, tool)
	}

	return tools, nil
}