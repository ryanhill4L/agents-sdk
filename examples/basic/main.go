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

// Example tool: simple calculator
func add(a, b int) int {
	return a + b
}

func greet(name string) string {
	return fmt.Sprintf("Hello, %s! Nice to meet you.", name)
}

func main() {
	fmt.Println("🤖 OpenAI Agents SDK - Basic Example")
	fmt.Println("===================================")

	// Create some tools
	addTool, err := tools.NewFunctionTool("add", "Adds two numbers together", add)
	if err != nil {
		log.Fatal("Failed to create add tool:", err)
	}

	greetTool, err := tools.NewFunctionTool("greet", "Greets a person by name", greet)
	if err != nil {
		log.Fatal("Failed to create greet tool:", err)
	}

	// Create agent with tools
	agent := agents.NewAgent("Assistant",
		agents.WithInstructions("You are a helpful assistant with access to tools."),
		agents.WithModel("gpt-4"),
		agents.WithTools(addTool, greetTool),
		agents.WithTemperature(0.7),
	)

	// Validate agent
	if err := agent.Validate(); err != nil {
		log.Fatal("Agent validation failed:", err)
	}

	fmt.Println("✅ Created agent:", agent.GetName())
	fmt.Println("📋 Instructions:", agent.GetInstructions())
	fmt.Println("🛠️  Tools available:", len(agent.Tools))

	// Create runner with console tracer for debugging
	runner := agents.NewRunner(
		agents.WithProvider(providers.NewOpenAIProvider()),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(5),
	)

	// Run a simple interaction
	ctx := context.Background()
	result, err := runner.Run(ctx, agent, "Hello! Can you add 5 and 3 for me?")
	if err != nil {
		log.Fatal("Failed to run agent:", err)
	}

	// Display results
	fmt.Println("\n🏁 Results:")
	fmt.Println("Final Output:", result.FinalOutput)
	fmt.Println("Total Turns:", result.Metrics.TotalTurns)
	fmt.Println("Total Tokens:", result.Metrics.TotalTokens)
	fmt.Println("Tool Calls:", result.Metrics.ToolCalls)
	fmt.Println("Duration:", result.Metrics.Duration)

	fmt.Println("\n💬 Conversation:")
	for i, msg := range result.Messages {
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, msg.Content)
	}
}
