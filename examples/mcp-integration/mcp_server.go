package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MCPRequest represents an incoming MCP request
type MCPRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error response
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ParameterSchema describes function parameters (matches the main package)
type ParameterSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
}

// PropertySchema describes a single parameter (matches the main package)
type PropertySchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Tool represents a tool available on this MCP server
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      ParameterSchema `json:"schema"`
}

// MCPServer implements a simple MCP server
type MCPServer struct {
	tools map[string]*Tool
}

// NewMCPServer creates a new MCP server
func NewMCPServer() *MCPServer {
	server := &MCPServer{
		tools: make(map[string]*Tool),
	}

	// Register built-in tools
	server.registerFileSystemTools()
	server.registerWeatherTool()
	server.registerTimeTools()

	return server
}

// registerFileSystemTools adds filesystem-related tools
func (s *MCPServer) registerFileSystemTools() {
	// List files tool
	listFilesTool := &Tool{
		Name:        "list_files",
		Description: "List files in a directory",
		Schema: ParameterSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"directory": {
					Type:        "string",
					Description: "Directory path to list files from",
				},
			},
			Required: []string{"directory"},
		},
	}
	s.tools["list_files"] = listFilesTool

	// Read file tool
	readFileTool := &Tool{
		Name:        "read_file",
		Description: "Read contents of a text file",
		Schema: ParameterSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"filepath": {
					Type:        "string",
					Description: "Path to the file to read",
				},
			},
			Required: []string{"filepath"},
		},
	}
	s.tools["read_file"] = readFileTool

	// Write file tool
	writeFileTool := &Tool{
		Name:        "write_file",
		Description: "Write content to a text file",
		Schema: ParameterSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"filepath": {
					Type:        "string",
					Description: "Path to the file to write",
				},
				"content": {
					Type:        "string",
					Description: "Content to write to the file",
				},
			},
			Required: []string{"filepath", "content"},
		},
	}
	s.tools["write_file"] = writeFileTool
}

// registerWeatherTool adds a mock weather tool
func (s *MCPServer) registerWeatherTool() {
	weatherTool := &Tool{
		Name:        "get_weather",
		Description: "Get current weather information for a city",
		Schema: ParameterSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"city": {
					Type:        "string",
					Description: "Name of the city to get weather for",
				},
				"units": {
					Type:        "string",
					Description: "Temperature units (celsius or fahrenheit)",
				},
			},
			Required: []string{"city"},
		},
	}
	s.tools["get_weather"] = weatherTool
}

// registerTimeTools adds time-related tools
func (s *MCPServer) registerTimeTools() {
	getCurrentTimeTool := &Tool{
		Name:        "get_current_time",
		Description: "Get the current date and time",
		Schema: ParameterSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"timezone": {
					Type:        "string",
					Description: "Timezone to get time for (optional, defaults to UTC)",
				},
			},
			Required: []string{},
		},
	}
	s.tools["get_current_time"] = getCurrentTimeTool
}

// ServeHTTP handles HTTP requests
func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		s.sendError(w, 405, "Method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.sendError(w, 400, "Failed to read request body")
		return
	}

	var request MCPRequest
	if err := json.Unmarshal(body, &request); err != nil {
		s.sendError(w, 400, "Invalid JSON request")
		return
	}

	s.handleRequest(w, &request)
}

// handleRequest processes an MCP request
func (s *MCPServer) handleRequest(w http.ResponseWriter, request *MCPRequest) {
	switch request.Method {
	case "tools/list":
		s.handleListTools(w)
	case "tools/get":
		s.handleGetTool(w, request.Params)
	case "tools/call":
		s.handleCallTool(w, request.Params)
	default:
		s.sendError(w, 400, fmt.Sprintf("Unknown method: %s", request.Method))
	}
}

// handleListTools returns the list of available tools
func (s *MCPServer) handleListTools(w http.ResponseWriter) {
	var tools []map[string]string
	for name := range s.tools {
		tools = append(tools, map[string]string{"name": name})
	}

	response := MCPResponse{
		Result: map[string]interface{}{
			"tools": tools,
		},
	}

	s.sendResponse(w, &response)
}

// handleGetTool returns information about a specific tool
func (s *MCPServer) handleGetTool(w http.ResponseWriter, params map[string]interface{}) {
	toolName, ok := params["name"].(string)
	if !ok {
		s.sendError(w, 400, "Missing tool name")
		return
	}

	tool, exists := s.tools[toolName]
	if !exists {
		s.sendError(w, 404, fmt.Sprintf("Tool not found: %s", toolName))
		return
	}

	response := MCPResponse{
		Result: tool,
	}

	s.sendResponse(w, &response)
}

// handleCallTool executes a tool with the provided arguments
func (s *MCPServer) handleCallTool(w http.ResponseWriter, params map[string]interface{}) {
	toolName, ok := params["name"].(string)
	if !ok {
		s.sendError(w, 400, "Missing tool name")
		return
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		s.sendError(w, 400, "Missing or invalid arguments")
		return
	}

	result, err := s.executeTool(toolName, arguments)
	if err != nil {
		s.sendError(w, 500, err.Error())
		return
	}

	response := MCPResponse{
		Result: result,
	}

	s.sendResponse(w, &response)
}

// executeTool executes the specified tool with arguments
func (s *MCPServer) executeTool(toolName string, args map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "list_files":
		return s.listFiles(args)
	case "read_file":
		return s.readFile(args)
	case "write_file":
		return s.writeFile(args)
	case "get_weather":
		return s.getWeather(args)
	case "get_current_time":
		return s.getCurrentTime(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// listFiles implements the list_files tool
func (s *MCPServer) listFiles(args map[string]interface{}) (interface{}, error) {
	directory, ok := args["directory"].(string)
	if !ok {
		return nil, fmt.Errorf("directory parameter is required")
	}

	// Basic security check - prevent directory traversal
	if strings.Contains(directory, "..") {
		return nil, fmt.Errorf("directory traversal not allowed")
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		files = append(files, map[string]interface{}{
			"name":  entry.Name(),
			"isDir": entry.IsDir(),
		})
	}

	return map[string]interface{}{
		"directory": directory,
		"files":     files,
	}, nil
}

// readFile implements the read_file tool
func (s *MCPServer) readFile(args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["filepath"].(string)
	if !ok {
		return nil, fmt.Errorf("filepath parameter is required")
	}

	// Basic security check
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("directory traversal not allowed")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return map[string]interface{}{
		"filepath": filePath,
		"content":  string(content),
		"size":     len(content),
	}, nil
}

// writeFile implements the write_file tool
func (s *MCPServer) writeFile(args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["filepath"].(string)
	if !ok {
		return nil, fmt.Errorf("filepath parameter is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required")
	}

	// Basic security check
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("directory traversal not allowed")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"filepath": filePath,
		"size":     len(content),
		"status":   "success",
	}, nil
}

// getWeather implements the get_weather tool (mock implementation)
func (s *MCPServer) getWeather(args map[string]interface{}) (interface{}, error) {
	city, ok := args["city"].(string)
	if !ok {
		return nil, fmt.Errorf("city parameter is required")
	}

	units := "celsius"
	if u, ok := args["units"].(string); ok {
		units = u
	}

	// Mock weather data
	weatherData := map[string]interface{}{
		"city":        city,
		"temperature": 22,
		"units":       units,
		"condition":   "Partly cloudy",
		"humidity":    65,
		"windSpeed":   "10 km/h",
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	if units == "fahrenheit" {
		weatherData["temperature"] = 72
	}

	return weatherData, nil
}

// getCurrentTime implements the get_current_time tool
func (s *MCPServer) getCurrentTime(args map[string]interface{}) (interface{}, error) {
	timezone := "UTC"
	if tz, ok := args["timezone"].(string); ok {
		timezone = tz
	}

	now := time.Now()
	if timezone != "UTC" {
		// For simplicity, we'll just use UTC. In a real implementation,
		// you'd load the timezone and convert accordingly.
		log.Printf("Timezone conversion not implemented, using UTC for %s", timezone)
	}

	return map[string]interface{}{
		"timestamp": now.Format(time.RFC3339),
		"timezone":  timezone,
		"unix":      now.Unix(),
		"formatted": now.Format("2006-01-02 15:04:05"),
	}, nil
}

// sendResponse sends a successful response
func (s *MCPServer) sendResponse(w http.ResponseWriter, response *MCPResponse) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendError sends an error response
func (s *MCPServer) sendError(w http.ResponseWriter, code int, message string) {
	response := MCPResponse{
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}

	w.WriteHeader(http.StatusOK) // MCP errors are still HTTP 200
	json.NewEncoder(w).Encode(response)
}

func main() {
	server := NewMCPServer()

	fmt.Println("🔧 MCP Server starting...")
	fmt.Println("Available tools:")
	for name, tool := range server.tools {
		fmt.Printf("  - %s: %s\n", name, tool.Description)
	}
	fmt.Println()
	fmt.Println("Server running on http://localhost:8080")
	fmt.Println("Use Ctrl+C to stop the server")

	http.Handle("/", server)
	log.Fatal(http.ListenAndServe(":8080", nil))
}