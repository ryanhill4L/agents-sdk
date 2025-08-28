package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

func main() {
	fmt.Println("🤖 Agents SDK - MCP Integration Example")
	fmt.Println("=====================================")
	fmt.Println()

	// Check if MCP server is running
	fmt.Println("🔍 Checking MCP server availability...")
	mcpClient := tools.NewMCPClient("http://localhost:8080")
	
	// Try to list tools to verify server is running
	toolNames, err := mcpClient.ListTools()
	if err != nil {
		fmt.Printf("❌ MCP server is not running or not accessible: %v\n", err)
		fmt.Println()
		fmt.Println("Please start the MCP server first:")
		fmt.Println("  go run mcp_server.go")
		fmt.Println()
		fmt.Println("Then in another terminal, run this example:")
		fmt.Println("  go run main.go")
		os.Exit(1)
	}

	fmt.Printf("✅ MCP server is running with %d tools available\n", len(toolNames))
	fmt.Println("Available tools:", toolNames)
	fmt.Println()

	// Create MCP tools
	fmt.Println("🔧 Creating MCP tools...")
	
	// Get specific tools we want to use
	listFilesTool, err := mcpClient.GetTool("list_files")
	if err != nil {
		log.Fatal("Failed to create list_files tool:", err)
	}

	readFileTool, err := mcpClient.GetTool("read_file")
	if err != nil {
		log.Fatal("Failed to create read_file tool:", err)
	}

	writeFileTool, err := mcpClient.GetTool("write_file")
	if err != nil {
		log.Fatal("Failed to create write_file tool:", err)
	}

	weatherTool, err := mcpClient.GetTool("get_weather")
	if err != nil {
		log.Fatal("Failed to create get_weather tool:", err)
	}

	timeTool, err := mcpClient.GetTool("get_current_time")
	if err != nil {
		log.Fatal("Failed to create get_current_time tool:", err)
	}

	fmt.Printf("✅ Created %d MCP tools successfully\n", 5)
	fmt.Println()

	// Create additional local tools for comparison
	calculateTool, err := tools.NewFunctionTool("calculate", "Performs basic arithmetic operations", calculate)
	if err != nil {
		log.Fatal("Failed to create calculate tool:", err)
	}

	// Create an agent with both MCP and local tools
	agent := agents.NewAgent("MCP Assistant",
		agents.WithInstructions(`You are a helpful assistant with access to both local functions and external MCP tools.

Available MCP Tools:
- list_files: List files in a directory
- read_file: Read contents of a text file
- write_file: Write content to a text file
- get_weather: Get current weather information for a city
- get_current_time: Get the current date and time

Available Local Tools:
- calculate: Perform basic arithmetic operations

Use the appropriate tools to help users with:
1. File system operations (listing, reading, writing files)
2. Weather information
3. Time queries
4. Basic calculations

Always explain what you're doing and provide helpful, detailed responses.`),
		agents.WithModel("gpt-4"),
		agents.WithTools(
			listFilesTool,
			readFileTool,
			writeFileTool,
			weatherTool,
			timeTool,
			calculateTool,
		),
	)

	// Create a provider
	provider, err := providers.NewOpenAIProviderFromEnv()
	if err != nil {
		log.Fatal("Failed to create provider (make sure OPENAI_API_KEY is set):", err)
	}

	// Create a runner with tracing
	runner := agents.NewRunner(
		agents.WithProvider(provider),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(10),
		agents.WithParallelTools(true),
	)

	// Example interactions demonstrating MCP capabilities
	examples := []struct {
		name    string
		query   string
		explain string
	}{
		{
			name:    "Weather Query",
			query:   "What's the weather like in San Francisco?",
			explain: "This will use the MCP get_weather tool to fetch weather data from the external server.",
		},
		{
			name:    "Time Query",
			query:   "What time is it right now?",
			explain: "This will use the MCP get_current_time tool to get the current timestamp.",
		},
		{
			name:    "File Operations",
			query:   "Create a file called 'mcp-demo.txt' with the content 'This file was created using MCP tools!' and then read it back to verify.",
			explain: "This will use MCP write_file and read_file tools to demonstrate file operations.",
		},
		{
			name:    "Directory Listing",
			query:   "List the files in the current directory '.'",
			explain: "This will use the MCP list_files tool to show directory contents.",
		},
		{
			name:    "Mixed Tools",
			query:   "Calculate 25 * 4, then tell me the current time, and create a file called 'calculation.txt' with the result and timestamp.",
			explain: "This will use both local tools (calculate) and MCP tools (get_current_time, write_file).",
		},
	}

	fmt.Println("🚀 Running MCP integration examples...")
	fmt.Println()

	ctx := context.Background()

	for i, example := range examples {
		fmt.Printf("📋 Example %d: %s\n", i+1, example.name)
		fmt.Printf("💬 Query: %s\n", example.query)
		fmt.Printf("📝 Expected: %s\n", example.explain)
		fmt.Println("⏳ Processing...")
		fmt.Println()

		start := time.Now()
		result, err := runner.Run(ctx, agent, example.query)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Response: %s\n", result.FinalOutput)
			fmt.Printf("📊 Metrics: %d turns, %d tool calls, %d tokens, %v\n",
				result.Metrics.TotalTurns,
				result.Metrics.ToolCalls,
				result.Metrics.TotalTokens,
				duration,
			)
		}

		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		// Add a small delay between examples
		time.Sleep(1 * time.Second)
	}

	fmt.Println("🎉 MCP integration examples completed!")
	fmt.Println()
	fmt.Println("Key Features Demonstrated:")
	fmt.Println("• External tool integration via MCP protocol")
	fmt.Println("• Dynamic tool discovery from MCP servers")
	fmt.Println("• Seamless mixing of local and remote tools")
	fmt.Println("• File system operations through MCP")
	fmt.Println("• Weather API integration")
	fmt.Println("• Time services")
	fmt.Println("• Error handling and validation")
}

// Local tool example for comparison
func calculate(operation, a, b string) (interface{}, error) {
	// Parse numbers
	var numA, numB float64
	if _, err := fmt.Sscanf(a, "%f", &numA); err != nil {
		return nil, fmt.Errorf("invalid number A: %s", a)
	}
	if _, err := fmt.Sscanf(b, "%f", &numB); err != nil {
		return nil, fmt.Errorf("invalid number B: %s", b)
	}

	// Perform operation
	var result float64
	switch operation {
	case "+", "add":
		result = numA + numB
	case "-", "subtract":
		result = numA - numB
	case "*", "multiply":
		result = numA * numB
	case "/", "divide":
		if numB == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = numA / numB
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	return map[string]interface{}{
		"operation": operation,
		"a":         numA,
		"b":         numB,
		"result":    result,
		"formatted": fmt.Sprintf("%.2f %s %.2f = %.2f", numA, operation, numB, result),
	}, nil
}