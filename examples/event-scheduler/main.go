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

	// Create the secure triage agent with guardrails
	triageAgent := localagents.NewSecureTriageAgent(database)

	// Validate agent
	if err := triageAgent.Validate(); err != nil {
		log.Fatal("Agent validation failed:", err)
	}

	// Create OpenAI provider
	provider, err := providers.NewOpenAIProviderFromEnv()
	if err != nil {
		log.Printf("Failed to create OpenAI provider: %v", err)
		log.Println("Make sure OPENAI_API_KEY environment variable is set")
		log.Println("export OPENAI_API_KEY='your-api-key-here'")
		return
	}

	// Create the runner
	runner := agents.NewRunner(
		agents.WithProvider(provider),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(5),
		agents.WithParallelTools(true),
	)

	// Interactive loop
	fmt.Println("🗓️  Event Scheduling Assistant Ready!")
	fmt.Println("=====================================")
	fmt.Println("Try asking:")
	fmt.Println("  - 'Show me all scheduled events'")
	fmt.Println("  - 'Find scheduling conflicts for users'")
	fmt.Println("  - 'Check for venue overlaps'")
	fmt.Println("  - 'What events is Alice attending?'")
	fmt.Println("\nType 'exit' to quit\n")

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
		result, err := runner.Run(ctx, triageAgent, input)
		if err != nil {
			log.Printf("❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("\n🤖 %s\n\n", result.FinalOutput)
		
		// Show some metrics
		if result.Metrics.ToolCalls > 0 || result.Metrics.Handoffs > 0 {
			fmt.Printf("📊 Metrics: %d turns, %d tool calls, %d handoffs, %d tokens, %v duration\n\n",
				result.Metrics.TotalTurns,
				result.Metrics.ToolCalls,
				result.Metrics.Handoffs,
				result.Metrics.TotalTokens,
				result.Metrics.Duration)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}

	fmt.Println("👋 Goodbye!")
}