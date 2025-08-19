package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	localagents "event-scheduler/agents"
	"event-scheduler/db"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

func main() {
	// Initialize database
	database, err := db.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Create the single event scheduler agent with all tools
	schedulerAgent := localagents.NewEventSchedulerAgent(database)

	// Validate agent
	if err := schedulerAgent.Validate(); err != nil {
		log.Fatal("Agent validation failed:", err)
	}

	// Create Anthropic provider
	provider, err := providers.NewAnthropicProviderFromEnv()
	if err != nil {
		log.Printf("Failed to create Anthropic provider: %v", err)
		log.Println("Make sure ANTHROPIC_API_KEY environment variable is set")
		log.Println("export ANTHROPIC_API_KEY='your-api-key-here'")

		// Return error - we need to set the ANTHROPIC_API_KEY environment variable
		log.Fatal("Please set the ANTHROPIC_API_KEY environment variable")
	}

	// Create the runner
	runner := agents.NewRunner(
		agents.WithProvider(provider),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(10),
	)

	// Interactive loop
	fmt.Println("🗓️  Event Scheduling Assistant Ready!")
	fmt.Println("=====================================")
	fmt.Println("Try asking:")
	fmt.Println("  - 'Show me all scheduled events'")
	fmt.Println("  - 'Find users with scheduling conflicts'")
	fmt.Println("  - 'Check for venue booking conflicts'")
	fmt.Println("  - 'What events is Alice attending?'")
	fmt.Println("  - 'List all events at the Conference Room'")
	fmt.Println("  - 'Show me events happening tomorrow'")
	fmt.Println("\nType 'exit' to quit")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			break
		}

		if input == "" {
			continue
		}

		// Run the agent
		ctx := context.Background()
		result, err := runner.Run(ctx, schedulerAgent, input)
		if err != nil {
			log.Printf("❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("\n🤖 %s\n\n", result.FinalOutput)

		// Show some metrics
		fmt.Printf("📊 Metrics: %d turns, %d tool calls, %d handoffs, %d tokens, %v duration\n",
			result.Metrics.TotalTurns,
			result.Metrics.ToolCalls,
			result.Metrics.Handoffs,
			result.Metrics.TotalTokens,
			result.Metrics.Duration)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}

	fmt.Println("👋 Goodbye!")
}
